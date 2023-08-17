package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
)

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
				backOffDuration := time.Duration((retries+1)*(retries+1)) * time.Second
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
