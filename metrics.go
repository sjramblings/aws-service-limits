package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
)

// var (
// 	timeframe           int
// 	servicecode         string
// 	cfg                 aws.Config
// 	outputFormat        string
// 	excludeNotAvailable bool
// 	listServices        bool
// )

func getMetricStatistics(namespace, class, resource, service, metricType, metricName, statistic string) (float64, error) {
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
	case "Sum":
		stat = types.StatisticSum
	default:
		return 0, fmt.Errorf("unsupported statistic: %s", statistic)
	}

	// Build the dimensions dynamically based on provided values
	var dimensions []types.Dimension
	if class != "" {
		dimensions = append(dimensions, types.Dimension{Name: aws.String("Class"), Value: aws.String(class)})
	}
	if resource != "" {
		dimensions = append(dimensions, types.Dimension{Name: aws.String("Resource"), Value: aws.String(resource)})
	}
	if service != "" {
		dimensions = append(dimensions, types.Dimension{Name: aws.String("Service"), Value: aws.String(service)})
	}
	if metricType != "" {
		dimensions = append(dimensions, types.Dimension{Name: aws.String("Type"), Value: aws.String(metricType)})
	}

	input := cloudwatch.GetMetricStatisticsInput{
		Namespace:  aws.String(namespace),
		Dimensions: dimensions,
		StartTime:  aws.Time(time.Now().Add(-1 * time.Duration(timeframe) * time.Hour)),
		EndTime:    aws.Time(time.Now()),
		Period:     aws.Int32(60),
		Statistics: []types.Statistic{stat},
		MetricName: aws.String(metricName),
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
	case types.StatisticSum:
		return *result.Datapoints[0].Sum, nil
	default:
		return 0, fmt.Errorf("unsupported statistic: %s", statistic)
	}
}
