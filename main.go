package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	sqTypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
)

const (
	accountWidth     = 12
	regionWidth      = 15
	serviceWidth     = 20
	globalQuotaWidth = 6
	valueWidth       = 15
	usageWidth       = 15
	nameWidth        = 80

	maxRetries = 5
)

var (
	timeframe           int
	servicecode         string
	cfg                 aws.Config
	outputFormat        string
	excludeNotAvailable bool
	listServices        bool
)

func init() {
	// Bind the command-line flag to the timeframe variable
	flag.IntVar(&timeframe, "timeframe", 1, "Timeframe for the CloudWatch query in hours. Options: 1, 24, 48, 72, etc.")

	// Bind the command-line flag to the serviceCode variable
	flag.StringVar(&servicecode, "servicecode", "ec2", "The AWS Service Code to query. Default is 'ec2'.")

	// Bind the command-line flag to the outputFormat variable
	flag.StringVar(&outputFormat, "format", "table", "Output format. Options: table, csv, markdown, json.")

	// Add the command-line flag for excluding "Not Available" usage values
	flag.BoolVar(&excludeNotAvailable, "exclude-na", false, "Exclude items with a usage value of 'Not Available'")

	// Add the command-line flag for listing all supported services
	flag.BoolVar(&listServices, "list-services", false, "List all the services supported by the AWS Service Quota API and exit.")

	// Customize the default usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Query AWS service quotas and usage.\n\n")
		fmt.Fprintf(os.Stderr, "The tool fetches and displays service quotas for AWS services.\n")
		fmt.Fprintf(os.Stderr, "\nGetting started:\n\n")
		fmt.Println("  # Fetch and display AWS service quotas for EC2")
		// Resolve these errors
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  aws-service-limits [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Available Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  aws-service-limits [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Available Flags:\n")
		flag.PrintDefaults()
	}

	var err error
	cfg, err = config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error: " + err.Error())
	}
}

func main() {
	flag.Parse()

	svc := servicequotas.NewFromConfig(cfg)

	if listServices {
		listSupportedServices()
		os.Exit(0)
	}

	input := &servicequotas.ListServiceQuotasInput{
		ServiceCode: aws.String(servicecode),
	}

	var wg sync.WaitGroup
	var completedTasks int32
	var totalTasks int32

	quotaInfos := make([]QuotaInfo, 0)
	mu := sync.Mutex{}

	progressTicker := time.NewTicker(1 * time.Second)
	defer progressTicker.Stop()

	go func() {
		for range progressTicker.C {
			if !isOutputRedirected() {
				fmt.Printf("\rCompleted %d/%d tasks", atomic.LoadInt32(&completedTasks), totalTasks)
			}
		}
	}()

	for {
		result, err := svc.ListServiceQuotas(context.TODO(), input)
		if err != nil {
			panic("failed to list service quotas: " + err.Error())
		}

		atomic.AddInt32(&totalTasks, int32(len(result.Quotas)))

		for _, quota := range result.Quotas {
			wg.Add(1)
			go func(q sqTypes.ServiceQuota) {
				defer wg.Done()

				processQuota(q, &quotaInfos, &mu)
				atomic.AddInt32(&completedTasks, 1)
			}(quota)
		}

		if result.NextToken == nil {
			break
		}
		input.NextToken = result.NextToken
	}

	wg.Wait()
	if !isOutputRedirected() {
		fmt.Println("\nAll tasks completed!")
	}

	// Filter out quotaInfos with "Not Available" usage if the flag is set
	if excludeNotAvailable {
		var filteredQuotaInfos []QuotaInfo
		for _, qi := range quotaInfos {
			if qi.Usage != "Not Available" {
				filteredQuotaInfos = append(filteredQuotaInfos, qi)
			}
		}
		quotaInfos = filteredQuotaInfos
	}

	// Sort the quotaInfos slice based on QuotaName
	sort.Slice(quotaInfos, func(i, j int) bool {
		return quotaInfos[i].QuotaName < quotaInfos[j].QuotaName
	})

	if outputFormat == "json" {
		b, err := json.MarshalIndent(quotaInfos, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating JSON output: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(string(b))
	} else if outputFormat == "csv" {
		writer := csv.NewWriter(os.Stdout)
		writer.Write([]string{"Account ID", "Region", "Service Code", "Quota Name", "Value", "Usage", "Global"})
		for _, qi := range quotaInfos {
			globalQuotaStr := fmt.Sprintf("%t", qi.GlobalQuota) // Convert the boolean to string
			writer.Write([]string{qi.AccountID, qi.Region, qi.ServiceCode, qi.QuotaName, qi.Value, qi.Usage, globalQuotaStr})
		}
		writer.Flush()
	} else {
		// Print the sorted quotaInfos
		printHeader()
		for _, qi := range quotaInfos {
			globalQuotaStr := fmt.Sprintf("%t", qi.GlobalQuota) // Convert the boolean to string
			printQuota(qi.AccountID, qi.Region, qi.ServiceCode, qi.QuotaName, qi.Value, qi.Usage, globalQuotaStr)
		}
	}
}
