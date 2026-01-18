package pipeline

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
)

func TestEventTypes(t *testing.T) {
	// Test that all event types are defined
	eventTypes := []EventType{
		EventStageStarted, EventStageFinished,
		EventWorkerStarted, EventWorkerFinished,
		EventItemProcessed, EventItemFailed,
		EventRouteStarted, EventRouteFinished,
		EventRouteError, EventUnsupportedType,
		EventNoMatch, EventConfigError,
		EventError, EventWarning,
		EventInfo, EventDebug,
	}

	for _, eventType := range eventTypes {
		if string(eventType) == "" {
			t.Errorf("Event type should not be empty: %v", eventType)
		}
	}
}

func TestObserverFunc(t *testing.T) {
	var receivedEvent Event
	observer := ObserverFunc(func(ctx context.Context, event Event) {
		receivedEvent = event
	})

	testEvent := Event{
		Type:    EventInfo,
		Message: "test message",
	}

	observer.OnEvent(context.Background(), testEvent)

	if receivedEvent.Type != EventInfo {
		t.Errorf("Expected event type %s, got %s", EventInfo, receivedEvent.Type)
	}
	if receivedEvent.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", receivedEvent.Message)
	}
}

func TestMultiObserver(t *testing.T) {
	var events1, events2 []Event

	observer1 := ObserverFunc(func(ctx context.Context, event Event) {
		events1 = append(events1, event)
	})

	observer2 := ObserverFunc(func(ctx context.Context, event Event) {
		events2 = append(events2, event)
	})

	multiObserver := NewMultiObserver(observer1, observer2)

	testEvent := Event{
		Type:    EventInfo,
		Message: "test",
	}

	multiObserver.OnEvent(context.Background(), testEvent)

	if len(events1) != 1 || events1[0].Type != EventInfo {
		t.Error("First observer did not receive event correctly")
	}
	if len(events2) != 1 || events2[0].Type != EventInfo {
		t.Error("Second observer did not receive event correctly")
	}
}

func TestNoOpImplementations(t *testing.T) {
	// Test that no-op implementations don't panic
	noOpObserver := NoOpObserver{}
	noOpObserver.OnEvent(context.Background(), Event{})

	noOpLogger := NoOpLogger{}
	noOpLogger.Debug("test")
	noOpLogger.Info("test")
	noOpLogger.Warn("test")
	noOpLogger.Error("test")
}

func TestSlogLogger(t *testing.T) {
	var output strings.Builder

	slogLogger := NewLoggerToWriter(&output, false, slog.LevelDebug)
	logger := NewSlogLogger(slogLogger)

	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "number", 42)
	logger.Warn("warning message")
	logger.Error("error message", "error", "some error")

	result := output.String()

	if !strings.Contains(result, "debug message") {
		t.Error("Slog logger should log debug messages when debug is enabled")
	}
	if !strings.Contains(result, "info message") {
		t.Error("Slog logger should log info messages")
	}
	if !strings.Contains(result, "key=value") {
		t.Error("Slog logger should include field key-value pairs")
	}
}

func TestJSONLogger(t *testing.T) {
	var output strings.Builder

	slogLogger := NewLoggerToWriter(&output, true, slog.LevelDebug)
	logger := NewSlogLogger(slogLogger)

	logger.Info("test message", "key", "value", "number", 42)

	result := output.String()

	if !strings.Contains(result, `"msg":"test message"`) {
		t.Error("JSON logger should include message in JSON format")
	}
	if !strings.Contains(result, `"level":"INFO"`) {
		t.Error("JSON logger should include level in JSON format")
	}
	if !strings.Contains(result, `"key":"value"`) {
		t.Error("JSON logger should include fields in JSON format")
	}
}

func TestLoggerObserver(t *testing.T) {
	var output strings.Builder
	slogLogger := NewLoggerToWriter(&output, false, slog.LevelDebug)
	logger := NewSlogLogger(slogLogger)
	observer := NewLoggerObserver(logger)

	// Test different event types map to correct log levels
	testCases := []struct {
		eventType    EventType
		expectedText string
	}{
		{EventError, "level=ERROR"},
		{EventWarning, "level=WARN"},
		{EventInfo, "level=INFO"},
		{EventDebug, "level=DEBUG"},
	}

	for _, tc := range testCases {
		output.Reset()
		event := Event{
			Type:    tc.eventType,
			Message: "test message",
		}
		observer.OnEvent(context.Background(), event)

		if !strings.Contains(output.String(), tc.expectedText) {
			t.Errorf("Event type %s should map to log level %s", tc.eventType, tc.expectedText)
		}
	}
}

func TestStageWithConfig(t *testing.T) {
	var events []Event
	observer := ObserverFunc(func(ctx context.Context, event Event) {
		events = append(events, event)
	})

	config := DefaultConfig().WithObserver(observer)

	// Create input
	input := make(chan int, 3)
	input <- 1
	input <- 2
	input <- 3
	close(input)

	// Process with config
	output := StartStageWithConfig("test_stage", 2, func(x int) int {
		return x * 2
	}, config, input)

	// Consume results
	var results []int
	for result := range output {
		results = append(results, result)
	}

	// Verify results
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Verify events were emitted
	stageStartedFound := false
	stageFinishedFound := false
	workerStartedCount := 0
	itemProcessedCount := 0

	for _, event := range events {
		if event.StageName != "test_stage" && event.Type != EventDebug {
			continue // Skip non-stage events
		}

		switch event.Type {
		case EventStageStarted:
			stageStartedFound = true
		case EventStageFinished:
			stageFinishedFound = true
		case EventWorkerStarted:
			workerStartedCount++
		case EventItemProcessed:
			itemProcessedCount++
		}
	}

	if !stageStartedFound {
		t.Error("Stage started event should be emitted")
	}
	if !stageFinishedFound {
		t.Error("Stage finished event should be emitted")
	}
	if workerStartedCount != 2 {
		t.Errorf("Expected 2 worker started events, got %d", workerStartedCount)
	}
	if itemProcessedCount != 3 {
		t.Errorf("Expected 3 item processed events, got %d", itemProcessedCount)
	}
}

func TestStageWithErrorAndConfig(t *testing.T) {
	var events []Event
	observer := ObserverFunc(func(ctx context.Context, event Event) {
		events = append(events, event)
	})

	config := DefaultConfig().WithObserver(observer)

	// Create input with some data that will cause errors
	input := make(chan int, 3)
	input <- 1 // will succeed
	input <- 0 // will cause error (division by zero simulation)
	input <- 2 // will succeed
	close(input)

	errorChan := make(chan error, 10)

	// Process with error handling
	output := StartStageWithErrAndConfig("error_test_stage", 1, func(x int) (int, error) {
		if x == 0 {
			return 0, errors.New("division by zero")
		}
		return 10 / x, nil
	}, errorChan, config, input)

	// Consume results and errors
	var results []int
	var processingErrors []error

	// Use goroutine to collect errors
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range errorChan {
			processingErrors = append(processingErrors, err)
		}
	}()

	// Collect results
	for result := range output {
		results = append(results, result)
	}

	close(errorChan)
	wg.Wait()

	// Verify we got 2 successful results and 1 error
	if len(results) != 2 {
		t.Errorf("Expected 2 successful results, got %d", len(results))
	}
	if len(processingErrors) != 1 {
		t.Errorf("Expected 1 processing error, got %d", len(processingErrors))
	}

	// Check for error events
	errorEventFound := false
	for _, event := range events {
		if event.Type == EventItemFailed {
			errorEventFound = true
			break
		}
	}
	if !errorEventFound {
		t.Error("Expected to find EventItemFailed in events")
	}
}

func TestRouterWithConfig(t *testing.T) {
	var events []Event
	observer := ObserverFunc(func(ctx context.Context, event Event) {
		events = append(events, event)
	})

	config := DefaultConfig().WithObserver(observer)

	// Create input
	input := make(chan int, 4)
	input <- 1
	input <- 2
	input <- 3
	input <- 4
	close(input)

	// Route even/odd
	evenChan, oddChan := ConditionalRouteWithConfig(
		input,
		func(x int) bool { return x%2 == 0 },
		2, 2, config,
	)

	// Collect results
	var evens, odds []int

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for even := range evenChan {
			evens = append(evens, even)
		}
	}()

	go func() {
		defer wg.Done()
		for odd := range oddChan {
			odds = append(odds, odd)
		}
	}()

	wg.Wait()

	// Verify routing worked
	if len(evens) != 2 {
		t.Errorf("Expected 2 even numbers, got %d", len(evens))
	}
	if len(odds) != 2 {
		t.Errorf("Expected 2 odd numbers, got %d", len(odds))
	}

	// Verify route events were emitted
	routeStartedFound := false
	routeFinishedFound := false

	for _, event := range events {
		if event.Type == EventRouteStarted {
			routeStartedFound = true
		}
		if event.Type == EventRouteFinished {
			routeFinishedFound = true
		}
	}

	if !routeStartedFound {
		t.Error("Route started event should be emitted")
	}
	if !routeFinishedFound {
		t.Error("Route finished event should be emitted")
	}
}

func TestRouterErrorHandling(t *testing.T) {
	var events []Event
	observer := ObserverFunc(func(ctx context.Context, event Event) {
		events = append(events, event)
	})

	config := DefaultConfig().WithObserver(observer)

	// Test configuration error
	input := make(chan int, 1)
	close(input)

	// Mismatched predicates and buffer sizes should return error
	outputs, err := RouteByPredicateWithConfig(
		input,
		[]int{1, 2}, // 2 buffer sizes
		config,
		func(x int) bool { return true }, // only 1 predicate
	)

	if err == nil {
		t.Error("Expected configuration error for mismatched predicates and buffer sizes")
	}
	if outputs != nil {
		t.Error("Outputs should be nil when configuration error occurs")
	}

	// Check for config error event
	configErrorFound := false
	for _, event := range events {
		if event.Type == EventConfigError {
			configErrorFound = true
			break
		}
	}
	if !configErrorFound {
		t.Error("Expected EventConfigError to be emitted")
	}
}

func TestMetricsCollector(t *testing.T) {
	config := DefaultConfig()
	metrics := NewMetricsCollector(config)

	// Simulate some events
	testEvents := []Event{
		{Type: EventStageStarted, StageName: "test_stage", Metadata: map[string]any{"workers": 2}},
		{Type: EventItemProcessed, StageName: "test_stage"},
		{Type: EventItemProcessed, StageName: "test_stage"},
		{Type: EventItemFailed, StageName: "test_stage"},
		{Type: EventStageFinished, StageName: "test_stage"},
	}

	for _, event := range testEvents {
		metrics.OnEvent(context.Background(), event)
	}

	// Get final metrics
	finalMetrics := metrics.GetMetrics()

	if finalMetrics.ItemsProcessed != 2 {
		t.Errorf("Expected 2 items processed, got %d", finalMetrics.ItemsProcessed)
	}
	if finalMetrics.ItemsFailed != 1 {
		t.Errorf("Expected 1 item failed, got %d", finalMetrics.ItemsFailed)
	}

	stageMetric, exists := finalMetrics.StageMetrics["test_stage"]
	if !exists {
		t.Error("Expected stage metrics for 'test_stage'")
	} else {
		if stageMetric.Workers != 2 {
			t.Errorf("Expected 2 workers in stage metrics, got %d", stageMetric.Workers)
		}
		if stageMetric.ItemsProcessed != 2 {
			t.Errorf("Expected 2 items processed in stage metrics, got %d", stageMetric.ItemsProcessed)
		}
		if stageMetric.ItemsFailed != 1 {
			t.Errorf("Expected 1 item failed in stage metrics, got %d", stageMetric.ItemsFailed)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Error("DefaultConfig should not return nil")
	}
	if config.Observer == nil {
		t.Error("Default config should have a non-nil observer")
	}
	if config.Logger == nil {
		t.Error("Default config should have a non-nil logger")
	}
	if config.Context == nil {
		t.Error("Default config should have a non-nil context")
	}
}

func TestConfigurationChaining(t *testing.T) {
	slogLogger := slog.Default()
	logger := NewSlogLogger(slogLogger)
	observer := NoOpObserver{}
	ctx := context.WithValue(context.Background(), "test", "value")

	config := DefaultConfig().
		WithLogger(logger).
		WithObserver(observer).
		WithContext(ctx)

	if config.Logger != logger {
		t.Error("WithLogger should set the logger")
	}
	if config.Observer != observer {
		t.Error("WithObserver should set the observer")
	}
	if config.Context != ctx {
		t.Error("WithContext should set the context")
	}
}

func TestEmitEventHelpers(t *testing.T) {
	var events []Event
	observer := ObserverFunc(func(ctx context.Context, event Event) {
		events = append(events, event)
	})

	config := DefaultConfig().
		WithObserver(observer)

	// Test EmitEvent
	EmitEvent(config, EventInfo, "test_stage", "test message", nil, map[string]any{"key": "value"})

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	event := events[0]
	if event.Type != EventInfo {
		t.Errorf("Expected EventInfo, got %s", event.Type)
	}
	if event.StageName != "test_stage" {
		t.Errorf("Expected stage name 'test_stage', got '%s'", event.StageName)
	}
	if event.Message != "test message" {
		t.Errorf("Expected message 'test message', got '%s'", event.Message)
	}
	if event.Metadata["key"] != "value" {
		t.Errorf("Expected metadata key 'value', got '%v'", event.Metadata["key"])
	}

	// Test EmitStageEvent
	events = nil // reset
	EmitStageEvent(config, EventWorkerStarted, "worker_stage", 5, "worker message", nil)

	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	stageEvent := events[0]
	if stageEvent.WorkerID != 5 {
		t.Errorf("Expected worker ID 5, got %d", stageEvent.WorkerID)
	}
}

func TestSlogLoggerWith(t *testing.T) {
	var output strings.Builder
	slogLogger := NewLoggerToWriter(&output, false, slog.LevelInfo)
	logger := NewSlogLogger(slogLogger)

	// Test With method
	contextLogger := logger.With("component", "test")
	contextLogger.Info("test message", "key", "value")

	result := output.String()

	if !strings.Contains(result, "component=test") {
		t.Error("With method should add context to subsequent log messages")
	}
	if !strings.Contains(result, "key=value") {
		t.Error("Additional fields should be included in log messages")
	}
}

// Integration test
func TestFullPipelineWithObservability(t *testing.T) {
	var output strings.Builder
	slogLogger := NewLoggerToWriter(&output, false, slog.LevelDebug)
	logger := NewSlogLogger(slogLogger)

	config := DefaultConfig().WithLogger(logger)
	metrics := NewMetricsCollector(config)

	multiObserver := NewMultiObserver(NewLoggerObserver(logger), metrics)
	config.WithObserver(multiObserver)

	// Create input
	input := make(chan int, 5)
	for i := 1; i <= 5; i++ {
		input <- i
	}
	close(input)

	// Multi-stage pipeline
	stage1 := StartStageWithConfig("multiply", 2, func(x int) int { return x * 2 }, config, input)
	stage2 := StartStageWithConfig("add", 2, func(x int) int { return x + 1 }, config, stage1)

	// Consume results
	var results []int
	for result := range stage2 {
		results = append(results, result)
	}

	// Verify results
	if len(results) != 5 {
		t.Errorf("Expected 5 results, got %d", len(results))
	}

	// Verify logging occurred
	logOutput := output.String()
	if !strings.Contains(logOutput, "multiply") {
		t.Error("Expected 'multiply' stage to be logged")
	}
	if !strings.Contains(logOutput, "add") {
		t.Error("Expected 'add' stage to be logged")
	}

	// Verify metrics
	finalMetrics := metrics.GetMetrics()
	if finalMetrics.ItemsProcessed != 10 { // 5 items × 2 stages
		t.Errorf("Expected 10 total items processed, got %d", finalMetrics.ItemsProcessed)
	}
	if len(finalMetrics.StageMetrics) != 2 {
		t.Errorf("Expected 2 stages in metrics, got %d", len(finalMetrics.StageMetrics))
	}
}
