package pipeline

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Example 1: Basic logging with console output
func ExampleBasicLogging() {
	fmt.Println("=== Example 1: Basic Logging ===")

	// Create a debug-enabled text logger
	logger := DefaultDebugTextLogger()

	// Create configuration
	config := DefaultConfigWithSlog(logger)

	// Create input data
	input := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		input <- i
	}
	close(input)

	// Process with observability
	output := StartStageWithConfig(
		"square_numbers",
		2, // 2 workers
		func(x int) int {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			return x * x
		},
		config,
		input,
	)

	// Consume results
	fmt.Println("Results:")
	for result := range output {
		fmt.Printf("  %d\n", result)
	}

	fmt.Println()
}

// Example 2: Error handling with observability
func ExampleErrorHandling() {
	fmt.Println("=== Example 2: Error Handling ===")

	logger := DefaultTextLogger() // Info level and above
	config := DefaultConfigWithSlog(logger)

	// Create input with some invalid data
	input := make(chan string, 6)
	input <- "10"
	input <- "invalid"
	input <- "20"
	input <- ""
	input <- "30"
	input <- "not_a_number"
	close(input)

	// Error channel
	errorChan := make(chan error, 10)

	// Process with error handling
	output := StartStageWithErrAndConfig(
		"parse_numbers",
		2,
		func(s string) (int, error) {
			if s == "" {
				return 0, errors.New("empty string")
			}
			num, err := strconv.Atoi(s)
			if err != nil {
				return 0, fmt.Errorf("failed to parse '%s': %w", s, err)
			}
			return num, nil
		},
		errorChan,
		config,
		input,
	)

	// Collect results and errors concurrently
	go func() {
		defer close(errorChan)
		for result := range output {
			fmt.Printf("  Parsed: %d\n", result)
		}
	}()

	fmt.Println("Results and Errors:")
	for err := range errorChan {
		fmt.Printf("  Error: %v\n", err)
	}

	fmt.Println()
}

// Example 3: Custom observer for metrics collection
func ExampleCustomObserver() {
	fmt.Println("=== Example 3: Custom Observer ===")

	// Custom observer that tracks processing statistics
	type StageStats struct {
		ItemsProcessed int
		WorkersStarted int
		StartTime      time.Time
		EndTime        time.Time
	}

	type StatsObserver struct {
		stageStats map[string]*StageStats
	}

	observer := &StatsObserver{
		stageStats: make(map[string]*StageStats),
	}

	observerFunc := func(ctx context.Context, event Event) {
		stats, exists := observer.stageStats[event.StageName]
		if !exists {
			stats = &StageStats{StartTime: time.Now()}
			observer.stageStats[event.StageName] = stats
		}

		switch event.Type {
		case EventItemProcessed:
			stats.ItemsProcessed++
		case EventWorkerStarted:
			stats.WorkersStarted++
		case EventStageFinished:
			stats.EndTime = time.Now()
		}
	}

	// Create configuration with custom observer
	config := DefaultConfig().WithObserver(ObserverFunc(observerFunc))

	// Create input
	input := make(chan string, 4)
	input <- "hello"
	input <- "world"
	input <- "pipeline"
	input <- "observability"
	close(input)

	// Process data
	output := StartStageWithConfig(
		"uppercase_strings",
		2,
		func(s string) string {
			time.Sleep(5 * time.Millisecond) // Simulate work
			return strings.ToUpper(s)
		},
		config,
		input,
	)

	// Consume results
	var results []string
	for result := range output {
		results = append(results, result)
	}

	// Print statistics
	for stageName, stats := range observer.stageStats {
		duration := stats.EndTime.Sub(stats.StartTime)
		fmt.Printf("Stage '%s' Statistics:\n", stageName)
		fmt.Printf("  Items Processed: %d\n", stats.ItemsProcessed)
		fmt.Printf("  Workers Started: %d\n", stats.WorkersStarted)
		fmt.Printf("  Duration: %v\n", duration)
		fmt.Printf("  Throughput: %.2f items/sec\n",
			float64(stats.ItemsProcessed)/duration.Seconds())
	}

	fmt.Printf("Final Results: %v\n\n", results)
}

// Example 4: Router with observability and error handling
func ExampleRouterObservability() {
	fmt.Println("=== Example 4: Router Observability ===")

	logger := DefaultDebugJSONLogger()
	config := DefaultConfigWithSlog(logger)

	// Create input with mixed data
	input := make(chan int, 8)
	for i := 1; i <= 8; i++ {
		input <- i
	}
	close(input)

	// Route based on even/odd with observability
	evenChan, oddChan := ConditionalRouteWithConfig(
		input,
		func(n int) bool { return n%2 == 0 },
		5, // even buffer size
		5, // odd buffer size
		config,
	)

	// Process even numbers
	evenSquares := StartStageWithConfig(
		"square_evens",
		1,
		func(n int) int { return n * n },
		config,
		evenChan,
	)

	// Process odd numbers
	oddCubes := StartStageWithConfig(
		"cube_odds",
		1,
		func(n int) int { return n * n * n },
		config,
		oddChan,
	)

	fmt.Println("Even squares:")
	for square := range evenSquares {
		fmt.Printf("  %d\n", square)
	}

	fmt.Println("Odd cubes:")
	for cube := range oddCubes {
		fmt.Printf("  %d\n", cube)
	}

	fmt.Println()
}

// Example 5: Multi-logger setup with metrics
func ExampleMultiLoggerWithMetrics() {
	fmt.Println("=== Example 5: Multi-Logger with Metrics ===")

	// Create multi-destination logger (both stdout and stderr)
	multiLogger := NewMultiLogger(true, slog.LevelInfo, os.Stdout, os.Stderr)

	// Create metrics collector
	config := DefaultConfigWithSlog(multiLogger)
	metricsCollector := NewMetricsCollector(config)

	// Combine logger observer and metrics collector
	loggerObserver := NewLoggerObserver(NewSlogLogger(multiLogger))
	multiObserver := NewMultiObserver(loggerObserver, metricsCollector)
	config.WithObserver(multiObserver)

	// Create a processing pipeline
	input := make(chan int, 10)
	for i := 1; i <= 10; i++ {
		input <- i
	}
	close(input)

	// Stage 1: Double the numbers
	doubled := StartStageWithConfig(
		"doubler",
		2,
		func(x int) int {
			time.Sleep(2 * time.Millisecond)
			return x * 2
		},
		config,
		input,
	)

	// Stage 2: Add 1 to each number
	incremented := StartStageWithConfig(
		"incrementer",
		3,
		func(x int) int {
			time.Sleep(1 * time.Millisecond)
			return x + 1
		},
		config,
		doubled,
	)

	// Consume all results
	var results []int
	for result := range incremented {
		results = append(results, result)
	}

	// Get final metrics
	finalMetrics := metricsCollector.GetMetrics()

	fmt.Printf("Pipeline completed!\n")
	fmt.Printf("Results: %v\n", results)
	fmt.Printf("Total items processed: %d\n", finalMetrics.ItemsProcessed)
	fmt.Printf("Total processing time: %v\n", finalMetrics.Duration)
	fmt.Printf("Average throughput: %.2f items/sec\n",
		float64(finalMetrics.ItemsProcessed)/finalMetrics.Duration.Seconds())

	fmt.Println("Stage breakdown:")
	for stageName, stage := range finalMetrics.StageMetrics {
		fmt.Printf("  %s: %d items with %d workers\n",
			stageName, stage.ItemsProcessed, stage.Workers)
	}

	fmt.Println()
}

// Example 6: Using slog with pipeline
func ExampleStandardLoggerIntegration() {
	fmt.Println("=== Example 6: Using slog ===")

	// Create an slog logger
	logger := DefaultDebugTextLogger()

	config := DefaultConfigWithSlog(logger)

	// Simple processing example
	input := make(chan string, 3)
	input <- "apple"
	input <- "banana"
	input <- "cherry"
	close(input)

	output := StartStageWithConfig(
		"fruit_processor",
		1,
		func(fruit string) string {
			return fmt.Sprintf("processed_%s", fruit)
		},
		config,
		input,
	)

	fmt.Println("Processed fruits:")
	for fruit := range output {
		fmt.Printf("  %s\n", fruit)
	}

	fmt.Println()
}

// Example 7: Error recovery in routing
func ExampleRouterErrorRecovery() {
	fmt.Println("=== Example 7: Router Error Recovery ===")

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn}))
	config := DefaultConfigWithSlog(logger)

	// Create input with mixed types (simulated as interface{})
	input := make(chan interface{}, 6)
	input <- "string_value"
	input <- 42
	input <- "another_string"
	input <- 3.14 // This will cause routing issues in type-specific routing
	input <- 100
	input <- "final_string"
	close(input)

	// Use predicate routing that handles mixed types gracefully
	outputs, err := RouteByPredicateWithConfig(
		input,
		[]int{3, 3}, // buffer sizes
		config,
		func(item interface{}) bool {
			_, isString := item.(string)
			return isString
		},
		func(item interface{}) bool {
			_, isInt := item.(int)
			return isInt
		},
	)

	if err != nil {
		fmt.Printf("Router setup error: %v\n", err)
		return
	}

	stringChan := outputs[0]
	intChan := outputs[1]

	// Process strings
	go func() {
		fmt.Println("String items:")
		for item := range stringChan {
			if str, ok := item.(string); ok {
				fmt.Printf("  String: %s\n", str)
			}
		}
	}()

	// Process integers
	fmt.Println("Integer items:")
	for item := range intChan {
		if num, ok := item.(int); ok {
			fmt.Printf("  Integer: %d\n", num)
		}
	}

	fmt.Println("Note: Non-matching items (like float64) were logged as warnings and skipped gracefully")
}

// Example 8: Performance monitoring with async observer
func ExampleAsyncObserver() {
	fmt.Println("=== Example 8: Async Performance Observer ===")

	// Async observer that doesn't block pipeline execution
	type AsyncPerfObserver struct {
		events   chan Event
		stopChan chan bool
		stats    map[string]int
	}

	perfObserver := &AsyncPerfObserver{
		events:   make(chan Event, 1000), // Large buffer
		stopChan: make(chan bool),
		stats:    make(map[string]int),
	}

	// Background processor for events
	go func() {
		for {
			select {
			case event := <-perfObserver.events:
				if event.Type == EventItemProcessed {
					perfObserver.stats[event.StageName]++
				}
			case <-perfObserver.stopChan:
				return
			}
		}
	}()

	// Observer implementation
	observerFunc := func(ctx context.Context, event Event) {
		select {
		case perfObserver.events <- event:
			// Event queued successfully
		default:
			// Buffer full, could log this condition
			fmt.Printf("Observer buffer full, dropping event: %s\n", event.Type)
		}
	}

	config := DefaultConfig().WithObserver(ObserverFunc(observerFunc))

	// Create larger dataset for performance testing
	input := make(chan int, 1000)
	for i := 0; i < 1000; i++ {
		input <- i
	}
	close(input)

	start := time.Now()

	// Multi-stage pipeline
	stage1 := StartStageWithConfig("multiply", 4, func(x int) int { return x * 2 }, config, input)
	stage2 := StartStageWithConfig("add", 4, func(x int) int { return x + 1 }, config, stage1)
	stage3 := StartStageWithConfig("modulo", 4, func(x int) int { return x % 100 }, config, stage2)

	// Consume results
	resultCount := 0
	for range stage3 {
		resultCount++
	}

	duration := time.Since(start)

	// Stop async observer and collect final stats
	time.Sleep(10 * time.Millisecond) // Allow async processing to catch up
	perfObserver.stopChan <- true

	fmt.Printf("Processed %d items in %v\n", resultCount, duration)
	fmt.Printf("Throughput: %.2f items/sec\n", float64(resultCount)/duration.Seconds())
	fmt.Println("Per-stage processing counts:")
	for stageName, count := range perfObserver.stats {
		fmt.Printf("  %s: %d items\n", stageName, count)
	}

	fmt.Print("\n")
}

// RunAllExamples runs all the examples in sequence
func RunAllExamples() {
	fmt.Println("Pipeline Observability Examples")
	fmt.Println("===============================")

	ExampleBasicLogging()
	ExampleErrorHandling()
	ExampleCustomObserver()
	ExampleMultiLoggerWithMetrics()
	ExampleStandardLoggerIntegration()

	fmt.Println("All examples completed!")
}
