package main

import (
	"context"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/servicequotas"
	sqTypes "github.com/aws/aws-sdk-go-v2/service/servicequotas/types"
)

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
