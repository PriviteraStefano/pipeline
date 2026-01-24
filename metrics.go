package pipeline

import (
	"context"
	"time"
)

type Identity struct {
	ID string `json:"id"`
}

type BaseMetrics struct {
	StartTime        time.Time     `json:"start_time"`
	EndTime          time.Time     `json:"end_time"`
	Duration         time.Duration `json:"duration"`
	ProcessedCounter int64
	FailedCounter    int64
}

// ExecutionMetrics holds metrics about pipeline execution
type ExecutionMetrics struct {
	BaseMetrics
	StagesMetrics  map[string]StageMetrics  `json:"stage_metrics"`
	RoutersMetrics map[string]RouterMetrics `json:"routers_metrics"`
}

// RouterMetrics represents metrics for a single router
type RouterMetrics struct {
	Identity
	BaseMetrics
	PathCounter map[string]int64 `json:"path_counter"`
	Errors      map[string]error `json:"errors"`
}

// StageMetrics represents metrics for a single stage
type StageMetrics struct {
	Identity
	BaseMetrics
	WorkersMetrics map[string]WorkerMetrics `json:"workers_metrics"`
}

// WorkerMetrics represents metrics for a single worker
type WorkerMetrics struct {
	Identity
	BaseMetrics
	ProcessesMetrics map[string]ProcessMetrics
}

type ProcessMetrics struct {
	Identity
	BaseMetrics
}

// MetricsCollector collects and tracks pipeline execution metrics
type MetricsCollector struct {
	metrics *ExecutionMetrics
	config  *Config
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(config *Config) *MetricsCollector {
	return &MetricsCollector{
		metrics: &ExecutionMetrics{
			StartTime:    time.Now(),
			StageMetrics: make(map[string]StageMetrics),
		},
		config: config,
	}
}

// OnEvent implements the Observer interface to collect metrics from events
func (mc *MetricsCollector) OnEvent(ctx context.Context, event Eventful) {
	switch event.GetType() {

	case EventItemProcessed:
		mc.metrics.ItemsProcessed++
		if stage, exists := mc.metrics.StageMetrics[event.StageName]; exists {
			stage.ItemsProcessed++
			mc.metrics.StageMetrics[event.StageName] = stage
		}

	case EventItemFailed:
		mc.metrics.ItemsFailed++
		if stage, exists := mc.metrics.StageMetrics[event.StageName]; exists {
			stage.ItemsFailed++
			mc.metrics.StageMetrics[event.StageName] = stage
		}

	case EventStageStarted:
		if workers, ok := event.Metadata["workers"].(int); ok {
			mc.metrics.StageMetrics[event.StageName] = StageMetrics{
				Name:    event.StageName,
				Workers: workers,
			}
		}
	}
}

// GetMetrics returns the current metrics and calculates the duration
func (mc *MetricsCollector) GetMetrics() ExecutionMetrics {
	mc.metrics.EndTime = time.Now()
	mc.metrics.Duration = mc.metrics.EndTime.Sub(mc.metrics.StartTime)
	return *mc.metrics
}

// Example helper functions for common pipeline patterns

// ProcessWithMetrics processes items through a pipeline and collects metrics
func ProcessWithMetrics[I, O any](
	name string,
	workers int,
	processor func(I) O,
	input <-chan I,
	logger Logger,
) (<-chan O, *MetricsCollector) {
	config := DefaultConfig().
		WithLogger(logger).
		WithContext(context.Background())

	metrics := NewMetricsCollector(config)
	multiObserver := NewMultiObserver(
		NewLoggerObserver(logger),
		metrics,
	)
	config.WithObserver(multiObserver)

	output := StartStageWithConfig(name, workers, processor, config, input)
	return output, metrics
}

// ProcessWithErrorHandling processes items with error handling and observability
func ProcessWithErrorHandling[I, O any](
	name string,
	workers int,
	processor func(I) (O, error),
	input <-chan I,
	logger Logger,
) (<-chan O, <-chan error, *MetricsCollector) {
	config := DefaultConfig().
		WithLogger(logger).
		WithContext(context.Background())

	metrics := NewMetricsCollector(config)
	multiObserver := NewMultiObserver(
		NewLoggerObserver(logger),
		metrics,
	)
	config.WithObserver(multiObserver)

	errorChan := make(chan error, workers*2)
	output := StartStageWithErrAndConfig(name, workers, processor, errorChan, config, input)

	return output, errorChan, metrics
}

// RouteWithLogging routes items with full observability
func RouteWithLogging[T any](
	input <-chan T,
	bufferSizes []int,
	logger Logger,
	predicates ...func(T) bool,
) ([]chan T, *MetricsCollector) {
	config := DefaultConfig().
		WithLogger(logger).
		WithContext(context.Background())

	metrics := NewMetricsCollector(config)
	multiObserver := NewMultiObserver(
		NewLoggerObserver(logger),
		metrics,
	)
	config.WithObserver(multiObserver)

	outputs, _ := RouteByPredicateWithConfig(input, bufferSizes, config, predicates...)
	return outputs, metrics
}
