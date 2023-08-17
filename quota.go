package main

import (
	"fmt"
	"strings"
	"sync"

	sqTypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
)

type QuotaInfo struct {
	AccountID   string
	Region      string
	ServiceCode string
	QuotaName   string
	Value       string
	Usage       string
	GlobalQuota bool
}

func processQuota(quota sqTypes.ServiceQuota, quotaInfos *[]QuotaInfo, mu *sync.Mutex) {
	arn := *quota.QuotaArn
	parts := strings.Split(arn, ":")
	region := parts[3]
	accountID := parts[4]
	unit := *quota.Unit
	valueFloat := *quota.Value
	valueInt := int(valueFloat)
	serviceCode := *quota.ServiceCode
	quotaCode := *quota.QuotaCode
	valueStr := fmt.Sprintf("%d", valueInt)

	// Check if the Unit value is present in the response
	if unit != "None" {
		valueStr = fmt.Sprintf("%s %s", valueStr, unit)
	}

	response, err := getUsageMetric(serviceCode, quotaCode)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	usage := "Not Available"
	if response.Quota != nil && response.Quota.UsageMetric != nil {
		metricNamespace := response.Quota.UsageMetric.MetricNamespace
		metricName := response.Quota.UsageMetric.MetricName
		metricDimensions := response.Quota.UsageMetric.MetricDimensions
		metricStatisticRecommendation := response.Quota.UsageMetric.MetricStatisticRecommendation

		class := metricDimensions["Class"]
		resource := metricDimensions["Resource"]
		service := metricDimensions["Service"]
		typeValue := metricDimensions["Type"]

		returnedUsage, err := getMetricStatistics(*metricNamespace, class, resource, service, typeValue, *metricName, *metricStatisticRecommendation)
		if err != nil {
			fmt.Println("Error retrieving metric statistics:", err.Error())
		} else {
			usage = fmt.Sprintf("%.0f", returnedUsage)
		}
	}

	globalQuota := quota.GlobalQuota // Extract the GlobalQuota value

	mu.Lock()
	*quotaInfos = append(*quotaInfos, QuotaInfo{
		AccountID:   accountID,
		Region:      region,
		ServiceCode: serviceCode,
		QuotaName:   *quota.QuotaName,
		Value:       valueStr,
		Usage:       usage,
		GlobalQuota: globalQuota, // Add the GlobalQuota field to the QuotaInfo struct
	})
	mu.Unlock()
}
