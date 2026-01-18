package pipeline

import (
	"context"
	"fmt"
	"log/slog"
	"os"
)

// Example_basic demonstrates basic pipeline usage with slog
func Example_basic() {
	// Create an slog logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create pipeline configuration
	config := DefaultConfigWithSlog(logger)

	// Create input channel
	input := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		input <- i
	}
	close(input)

	// Process with observability
	output := StartStageWithConfig(
		"multiply",
		2, // workers
		func(x int) int { return x * 2 },
		config,
		input,
	)

	// Consume results
	for result := range output {
		fmt.Println(result)
	}
}

// Example_errorHandling demonstrates error handling with observability
func Example_errorHandling() {
	logger := slog.Default()
	config := DefaultConfigWithSlog(logger)

	input := make(chan string, 3)
	input <- "10"
	input <- "invalid"
	input <- "20"
	close(input)

	errorChan := make(chan error, 10)

	// Process with error handling
	output := StartStageWithErrAndConfig(
		"parse",
		1,
		func(s string) (int, error) {
			var num int
			_, err := fmt.Sscanf(s, "%d", &num)
			return num, err
		},
		errorChan,
		config,
		input,
	)

	// Collect results
	go func() {
		for result := range output {
			fmt.Printf("Parsed: %d\n", result)
		}
		close(errorChan)
	}()

	// Handle errors
	for err := range errorChan {
		fmt.Printf("Error: %v\n", err)
	}
}

// Example_customObserver demonstrates custom observer for metrics
func Example_customObserver() {
	type Stats struct {
		itemsProcessed int
	}

	stats := &Stats{}

	// Custom observer
	observer := ObserverFunc(
		func(ctx context.Context, event Event) {
			if event.Type == EventItemProcessed {
				stats.itemsProcessed++
			}
		},
	)

	config := DefaultConfig().WithObserver(observer)

	input := make(chan int, 3)
	input <- 1
	input <- 2
	input <- 3
	close(input)

	output := StartStageWithConfig(
		"process",
		1,
		func(x int) int { return x * 2 },
		config,
		input,
	)

	// Consume all results
	for range output {
	}

	fmt.Printf("Processed %d items\n", stats.itemsProcessed)
	// Output: Processed 3 items
}

// Example_routing demonstrates graceful routing with error handling
func Example_routing() {
	logger := slog.Default()
	config := DefaultConfigWithSlog(logger)

	input := make(chan int, 6)
	for i := 1; i <= 6; i++ {
		input <- i
	}
	close(input)

	// Route even/odd numbers
	evenChan, oddChan := ConditionalRouteWithConfig(
		input,
		func(n int) bool { return n%2 == 0 },
		3, 3, // buffer sizes
		config,
	)

	fmt.Println("Even numbers:")
	for n := range evenChan {
		fmt.Println(n)
	}

	fmt.Println("Odd numbers:")
	for n := range oddChan {
		fmt.Println(n)
	}
}

// Example_metricsCollection demonstrates built-in metrics collection
func Example_metricsCollection() {
	logger := slog.Default()
	config := DefaultConfigWithSlog(logger)

	// Create metrics collector
	metrics := NewMetricsCollector(config)

	// Combine logger and metrics observers
	observer := NewMultiObserver(
		NewLoggerObserver(NewSlogLogger(logger)),
		metrics,
	)
	config.WithObserver(observer)

	input := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		input <- i
	}
	close(input)

	output := StartStageWithConfig(
		"process",
		2,
		func(x int) int { return x * 2 },
		config,
		input,
	)

	// Consume results
	for range output {
	}

	// Get final metrics
	finalMetrics := metrics.GetMetrics()
	fmt.Printf("Items processed: %d\n", finalMetrics.ItemsProcessed)
	fmt.Printf("Items failed: %d\n", finalMetrics.ItemsFailed)
}
