package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"flag"
	"sync"
	"sync/atomic"
	"sort"
	"os"
	"encoding/json"
	"encoding/csv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	sqTypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
	"github.com/fatih/color"
)

const (
	accountWidth = 15
	regionWidth  = 15
	serviceWidth = 15
	nameWidth    = 80
	valueWidth   = 6
	usageWidth   = 6
	maxRetries   = 5
)

var (
	timeframe int
	servicecode string
	cfg       aws.Config
	outputFormat string
	excludeNotAvailable	bool
	listServices bool

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
		fmt.Println("  aws-service-limits\n")
		fmt.Println("  # Fetch and display AWS service quotas for a specific AWS service (e.g., S3)")
		fmt.Println("  aws-service-limits --servicecode s3\n")
		fmt.Println("  # List all the services supported by the AWS Service Quota API")
		fmt.Println("  aws-service-limits --list-services\n")
		fmt.Println("  # Get help for a flag")
		fmt.Println("  aws-service-limits --help\n")
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

type QuotaInfo struct {
	AccountID   string
	Region      string
	ServiceCode string
	QuotaName   string
	Value       string
	Usage       string
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
			fmt.Printf("\rCompleted %d/%d tasks", atomic.LoadInt32(&completedTasks), totalTasks)
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
	fmt.Println("\nAll tasks completed!")

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
		writer.Write([]string{"Account ID", "Region", "Service Code", "Quota Name", "Value", "Usage"})
		for _, qi := range quotaInfos {
			writer.Write([]string{qi.AccountID, qi.Region, qi.ServiceCode, qi.QuotaName, qi.Value, qi.Usage})
		}
		writer.Flush()
	} else {
		// Print the sorted quotaInfos
		printHeader()
		for _, qi := range quotaInfos {
			printQuota(qi.AccountID, qi.Region, qi.ServiceCode, qi.QuotaName, qi.Value, qi.Usage)
		}
	}	
}

func processQuota(quota sqTypes.ServiceQuota, quotaInfos *[]QuotaInfo, mu *sync.Mutex) {
	arn := *quota.QuotaArn
	parts := strings.Split(arn, ":")
	region := parts[3]
	accountID := parts[4]
	valueFloat := *quota.Value
	valueInt := int(valueFloat)
	serviceCode := *quota.ServiceCode
	quotaCode := *quota.QuotaCode
	valueStr := fmt.Sprintf("%d", valueInt)

	response, err := getUsageMetric(serviceCode, quotaCode)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	usage := "Not Available"
	if response.Quota != nil && response.Quota.UsageMetric != nil {
		metricNamespace := response.Quota.UsageMetric.MetricNamespace
		metricDimensions := response.Quota.UsageMetric.MetricDimensions
		metricStatisticRecommendation := response.Quota.UsageMetric.MetricStatisticRecommendation

		class := metricDimensions["Class"]
		resource := metricDimensions["Resource"]
		service := metricDimensions["Service"]
		typeValue := metricDimensions["Type"]

		returnedUsage, err := getMetricStatistics(*metricNamespace, class, resource, service, typeValue, *metricStatisticRecommendation)
		if err != nil {
			fmt.Println("Error retrieving metric statistics:", err.Error())
		} else {
			usage = fmt.Sprintf("%.0f", returnedUsage)
		}
	}

	mu.Lock()
	*quotaInfos = append(*quotaInfos, QuotaInfo{
		AccountID:   accountID,
		Region:      region,
		ServiceCode: serviceCode,
		QuotaName:   *quota.QuotaName,
		Value:       valueStr,
		Usage:       usage,
	})
	mu.Unlock()
}

func getMetricStatistics(namespace, class, resource, service, metricType, statistic string) (float64, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return 0, fmt.Errorf("configuration error, %v", err)
	}

	client := cloudwatch.NewFromConfig(cfg)

	// Convert the string statistic to the appropriate SDK type
	var stat types.Statistic
	switch statistic {
	case "Average":
		stat = types.StatisticAverage
	case "Maximum":
		stat = types.StatisticMaximum
	// Add cases for other statistics as needed
	default:
		return 0, fmt.Errorf("unsupported statistic: %s", statistic)
	}

	input := cloudwatch.GetMetricStatisticsInput{
		Namespace: aws.String(namespace),
		Dimensions: []types.Dimension{
			{Name: aws.String("Class"), Value: aws.String(class)},
			{Name: aws.String("Resource"), Value: aws.String(resource)},
			{Name: aws.String("Service"), Value: aws.String(service)},
			{Name: aws.String("Type"), Value: aws.String(metricType)},
		},
		StartTime:  aws.Time(time.Now().Add(-1 * time.Duration(timeframe) * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int32(60),
		Statistics: []types.Statistic{stat},
		MetricName: aws.String("ResourceCount"),
	}

	result, err := client.GetMetricStatistics(context.TODO(), &input)
	if err != nil {
		return 0, err
	}

	if len(result.Datapoints) == 0 {
		return 0, nil
	}

	// Return the appropriate statistic value based on the input
	switch stat {
	case types.StatisticAverage:
		return *result.Datapoints[0].Average, nil
	case types.StatisticMaximum:
		return *result.Datapoints[0].Maximum, nil
	// Add cases for other statistics as needed
	default:
		return 0, fmt.Errorf("unsupported statistic: %s", statistic)
	}
}

// Retrieve the UsageMetric for a given ServiceCode and QuotaCode
func getUsageMetric(serviceCode, quotaCode string) (*servicequotas.GetAWSDefaultServiceQuotaOutput, error) {
    // Load the AWS SDK config
    cfg, err := config.LoadDefaultConfig(context.TODO())
    if err != nil {
        return nil, fmt.Errorf("configuration error: %v", err)
    }

    // Create a new Service Quotas client
    client := servicequotas.NewFromConfig(cfg)

    // Prepare the input for the GetAWSDefaultServiceQuota API call
    input := &servicequotas.GetAWSDefaultServiceQuotaInput{
        ServiceCode: aws.String(serviceCode),
        QuotaCode:   aws.String(quotaCode),
    }

    var result *servicequotas.GetAWSDefaultServiceQuotaOutput

    for retries := 0; retries < maxRetries; retries++ {
        result, err = client.GetAWSDefaultServiceQuota(context.TODO(), input)
        if err != nil {
            // Check for TooManyRequestsException and back off if necessary
            if strings.Contains(err.Error(), "TooManyRequestsException") {
                backOffDuration := time.Duration((retries + 1) * (retries + 1)) * time.Second
                time.Sleep(backOffDuration)
                continue
            } else {
                return nil, err
            }
        } else {
            break
        }
    }

    return result, err
}

func printHeader() {
	switch outputFormat {
	case "csv":
		fmt.Println("Account ID,Region,Service Code,Quota Name,Value,Usage")
	case "markdown":
		fmt.Println("| Account ID | Region | Service Code | Quota Name | Value | Usage |")
		fmt.Println("|------------|-------|--------------|------------|-------|-------|")
	case "table", "json": // For table and json, we do nothing in the header.
	default:
		fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", outputFormat)
		os.Exit(1)
	}
}

func printQuota(accountID, region, serviceCode, QuotaName, Value, Usage string) {
	// Define orange color
	orange := color.New(color.FgYellow).SprintFunc()

	switch outputFormat {
	case "csv":
		fmt.Printf("%s,%s,%s,%s,%s,%s\n", accountID, region, serviceCode, QuotaName, Value, Usage)
	case "markdown":
		fmt.Printf("| %s | %s | %s | %s | %s | %s |\n", accountID, region, serviceCode, QuotaName, Value, Usage)
	case "table":
		if Usage == "Not Available" {
			// If Usage is "Not Available", print the row in orange
			fmt.Printf("%s\n", orange(fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s", accountWidth, accountID, regionWidth, region, serviceWidth, serviceCode, nameWidth, QuotaName, valueWidth, Value, usageWidth, Usage)))
		} else {
			fmt.Printf("%-*s %-*s %-*s %-*s %-*s %-*s\n", accountWidth, accountID, regionWidth, region, serviceWidth, serviceCode, nameWidth, QuotaName, valueWidth, Value, usageWidth, Usage)
		}
	case "json":
		// For json, we'll handle the output in the main function.
	default:
		fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", outputFormat)
		os.Exit(1)
	}
}

func listSupportedServices() {
	var wg sync.WaitGroup
	nextToken := (*string)(nil) // Start with no token for the first request
	services := make([]sqTypes.ServiceInfo, 0)
	var mu sync.Mutex // Mutex to guard against concurrent access to the `services` slice

	progressTicker := time.NewTicker(1 * time.Second)
	defer progressTicker.Stop()

	go func() {
		for range progressTicker.C {
			// Update this to show progress for fetching services.
			// We don't have totalTasks here, so we'll just show a spinner or a simple message.
			fmt.Printf("\rFetching services...")
		}
	}()

	for {
		wg.Add(1)
		go func(nextToken *string) {
			defer wg.Done()
			svc := servicequotas.NewFromConfig(cfg)
			maxResults := int32(100)
			input := &servicequotas.ListServicesInput{
				MaxResults: &maxResults,
				NextToken:  nextToken,
			}
			result, err := svc.ListServices(context.TODO(), input)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching services: %v\n", err)
				return
			}
			mu.Lock()
			services = append(services, result.Services...)
			mu.Unlock()
			nextToken = result.NextToken
		}(nextToken)

		wg.Wait()
		if nextToken == nil {
			break
		}
	}

	// Sort the services based on ServiceName before displaying
	sort.Slice(services, func(i, j int) bool {
		return *services[i].ServiceName < *services[j].ServiceName
	})

	for _, service := range services {
		fmt.Printf("%s (%s)\n", *service.ServiceName, *service.ServiceCode)
	}
}
