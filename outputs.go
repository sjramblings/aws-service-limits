package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

const (
	OutputFormatCSV      = "csv"
	OutputFormatMarkdown = "markdown"
	OutputFormatTable    = "table"
	OutputFormatJSON     = "json"
)

func printHeader() {
	switch outputFormat {
	case OutputFormatCSV:
		fmt.Println("Account ID,Region,Service Code,Global,Value,Usage,Quota Name")
	case OutputFormatMarkdown:
		fmt.Println("| Account ID | Region | Service Code | Global | Value | Usage | Quota Name |")
		fmt.Println("|------------|--------|--------------|-------|-------|-------|------------|")
	case OutputFormatTable:
		fmt.Printf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s\n", accountWidth, "Account ID", regionWidth, "Region", serviceWidth, "Service", globalQuotaWidth, "Global", valueWidth, "Value", usageWidth, "Usage", nameWidth, "Quota Name")
	case OutputFormatJSON:
		// Print header for JSON format
	default:
		fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", outputFormat)
		os.Exit(1)
	}
}

func printQuota(accountID, region, serviceCode, QuotaName, Value, Usage, GlobalQuota string) {
	// Define orange color
	orange := color.New(color.FgYellow).SprintFunc()

	switch outputFormat {
	case OutputFormatCSV:
		fmt.Printf("%s,%s,%s,%s,%s,%s,%s\n", accountID, region, serviceCode, GlobalQuota, Value, Usage, QuotaName)
	case OutputFormatMarkdown:
		fmt.Printf("| %s | %s | %s | %s | %s | %s | %s |\n", accountID, region, serviceCode, GlobalQuota, Value, Usage, QuotaName)
	case OutputFormatTable:
		if Usage == "Not Available" {
			// If Usage is "Not Available", print the row in orange
			fmt.Printf("%s\n", orange(fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s", accountWidth, accountID, regionWidth, region, serviceWidth, serviceCode, globalQuotaWidth, GlobalQuota, valueWidth, Value, usageWidth, Usage, nameWidth, QuotaName)))
		} else {
			fmt.Printf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s\n", accountWidth, accountID, regionWidth, region, serviceWidth, serviceCode, globalQuotaWidth, GlobalQuota, valueWidth, Value, usageWidth, Usage, nameWidth, QuotaName)
		}
	case OutputFormatJSON:
		// Print quota for JSON format
	default:
		fmt.Fprintf(os.Stderr, "Unsupported format: %s\n", outputFormat)
		os.Exit(1)
	}
}

func isOutputRedirected() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		panic(err)
	}
	return (fileInfo.Mode() & os.ModeCharDevice) == 0
}
