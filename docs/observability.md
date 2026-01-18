# Pipeline Observability Guide

This document describes the observer and logging patterns available in the pipeline library.

## Overview

The pipeline library provides comprehensive observability through:

- **Observer Pattern**: Event-driven observability for pipeline operations
- **Pluggable Logging**: Structured logging with multiple backend support
- **Metrics Collection**: Built-in metrics collection and reporting
- **Error Handling**: Graceful error handling instead of panics

## Quick Start

### Basic Logging

```go
package main

import (
    "pipeline"
)

func main() {
    // Create a console logger
    logger := pipeline.NewConsoleLogger().WithDebug(true)
    
    // Create configuration with logger
    config := pipeline.DefaultConfig().WithLogger(logger)
    
    // Create input channel
    input := make(chan int, 10)
    for i := 0; i < 10; i++ {
        input <- i
    }
    close(input)
    
    // Process with observability
    output := pipeline.StartStageWithConfig(
        "multiply",
        2,
        func(x int) int { return x * 2 },
        config,
        input,
    )
    
    // Consume results
    for result := range output {
        println("Result:", result)
    }
}
```

### Custom Observer

```go
// Custom observer that counts events
type EventCounter struct {
    counts map[pipeline.EventType]int
}

func (ec *EventCounter) OnEvent(ctx context.Context, event pipeline.Event) {
    if ec.counts == nil {
        ec.counts = make(map[pipeline.EventType]int)
    }
    ec.counts[event.Type]++
    
    if event.Type == pipeline.EventItemProcessed {
        fmt.Printf("Processed item in stage: %s\n", event.StageName)
    }
}

// Usage
counter := &EventCounter{}
config := pipeline.DefaultConfig().WithObserver(counter)
```

## Event Types

### Stage Events

| Event Type | Description | When Triggered |
|------------|-------------|----------------|
| `EventStageStarted` | Stage begins execution | Stage initialization |
| `EventStageFinished` | Stage completes | All workers finish |
| `EventWorkerStarted` | Worker goroutine starts | Worker initialization |
| `EventWorkerFinished` | Worker goroutine ends | Worker cleanup |
| `EventItemProcessed` | Item successfully processed | After successful processing |
| `EventItemFailed` | Item processing failed | After processing error |

### Router Events

| Event Type | Description | When Triggered |
|------------|-------------|----------------|
| `EventRouteStarted` | Router begins operation | Router initialization |
| `EventRouteFinished` | Router completes | Router cleanup |
| `EventRouteError` | Router configuration error | Invalid configuration |
| `EventUnsupportedType` | Type not supported | Unknown type encountered |
| `EventNoMatch` | No routing rule matched | No predicate/key match |

### General Events

| Event Type | Description |
|------------|-------------|
| `EventError` | General error |
| `EventWarning` | Warning condition |
| `EventInfo` | Informational message |
| `EventDebug` | Debug information |

## Built-in Loggers

### Console Logger

```go
logger := pipeline.NewConsoleLogger().
    WithDebug(true).
    WithColors(true).
    WithOutput(os.Stdout).
    WithErrorOutput(os.Stderr)
```

Features:
- Colored output (configurable)
- Timestamp formatting
- Separate output streams for errors
- Debug level filtering

### JSON Logger

```go
logger := pipeline.NewJSONLogger(os.Stdout).WithDebug(true)
```

Features:
- Structured JSON output
- Machine-readable format
- Debug level filtering
- Custom output destination

### Standard Logger

```go
stdLog := log.New(os.Stdout, "[PIPELINE] ", log.LstdFlags)
logger := pipeline.NewStdLogger(stdLog).WithDebug(true)
```

Features:
- Uses Go's standard `log` package
- Configurable prefixes and flags
- Debug level filtering

### Multi Logger

```go
console := pipeline.NewConsoleLogger()
json := pipeline.NewJSONLogger(logFile)
logger := pipeline.NewMultiLogger(console, json)
```

Features:
- Writes to multiple destinations
- Combines different logger types
- Parallel logging

### Level Filter Logger

```go
logger := pipeline.NewConsoleLogger()
filtered := pipeline.NewLevelFilterLogger(logger, pipeline.InfoLevel)
```

Features:
- Filters messages below minimum level
- Wraps any other logger
- Runtime level adjustment

## Advanced Usage

### Custom Event Metadata

```go
observer := pipeline.ObserverFunc(func(ctx context.Context, event pipeline.Event) {
    if event.Metadata != nil {
        if userID, ok := event.Metadata["user_id"].(string); ok {
            fmt.Printf("Processing for user: %s\n", userID)
        }
    }
})
```

### Metrics Collection

```go
// Create metrics collector
config := pipeline.DefaultConfig()
metrics := pipeline.NewMetricsCollector(config)

// Combine with logging
logger := pipeline.NewConsoleLogger()
multiObserver := pipeline.NewMultiObserver(
    pipeline.NewLoggerObserver(logger),
    metrics,
)

config.WithObserver(multiObserver)

// Use the configuration
output := pipeline.StartStageWithConfig("stage1", 4, processor, config, input)

// Get metrics after processing
finalMetrics := metrics.GetMetrics()
fmt.Printf("Processed %d items in %v\n", 
    finalMetrics.ItemsProcessed, 
    finalMetrics.Duration)
```

### Context-Aware Observability

```go
ctx := context.WithValue(context.Background(), "request_id", "req-123")
config := pipeline.DefaultConfig().WithContext(ctx)

observer := pipeline.ObserverFunc(func(ctx context.Context, event pipeline.Event) {
    if reqID := ctx.Value("request_id"); reqID != nil {
        logger.Info("Event occurred", 
            pipeline.F("request_id", reqID),
            pipeline.F("event_type", event.Type),
        )
    }
})

config.WithObserver(observer)
```

### Error Handling in Routers

```go
// Before: Panicked on unsupported types
outputs := pipeline.Fork[string, int](input, 10, 10)

// After: Handles errors gracefully with observability
outputs, err := pipeline.ForkWithConfig[string, int](input, 10, 10, config)
if err != nil {
    logger.Error("Fork operation failed", pipeline.F("error", err))
    return
}
```

## Integration with External Systems

### Prometheus Metrics

```go
type PrometheusObserver struct {
    itemsProcessed *prometheus.CounterVec
    processingTime *prometheus.HistogramVec
}

func (p *PrometheusObserver) OnEvent(ctx context.Context, event pipeline.Event) {
    switch event.Type {
    case pipeline.EventItemProcessed:
        p.itemsProcessed.WithLabelValues(event.StageName).Inc()
    case pipeline.EventStageFinished:
        if duration, ok := event.Metadata["duration"].(time.Duration); ok {
            p.processingTime.WithLabelValues(event.StageName).
                Observe(duration.Seconds())
        }
    }
}
```

### Structured Logging (Logrus/Zap)

```go
// Logrus adapter
type LogrusAdapter struct {
    logger *logrus.Logger
}

func (l *LogrusAdapter) Info(msg string, fields ...pipeline.Field) {
    entry := l.logger.WithFields(logrus.Fields{})
    for _, field := range fields {
        entry = entry.WithField(field.Key, field.Value)
    }
    entry.Info(msg)
}

// Usage
logrusLogger := logrus.New()
adapter := &LogrusAdapter{logger: logrusLogger}
config := pipeline.DefaultConfig().WithLogger(adapter)
```

## Performance Considerations

### Observer Performance

- Observers are called synchronously in the pipeline goroutines
- Keep observer logic lightweight to avoid blocking pipeline execution
- Use buffered channels for expensive operations:

```go
type AsyncObserver struct {
    events chan pipeline.Event
}

func (a *AsyncObserver) OnEvent(ctx context.Context, event pipeline.Event) {
    select {
    case a.events <- event:
        // Event queued
    default:
        // Buffer full, drop event or handle differently
    }
}
```

### Logger Performance

- JSON logger is faster for machine processing
- Console logger is better for human reading
- Use level filtering to reduce overhead:

```go
// Only log warnings and errors in production
logger := pipeline.NewLevelFilterLogger(
    pipeline.NewJSONLogger(logFile),
    pipeline.WarnLevel,
)
```

## Best Practices

### 1. Always Use Configuration

```go
// Good: Explicit configuration
config := pipeline.DefaultConfig().WithLogger(logger)
output := pipeline.StartStageWithConfig(name, workers, fn, config, input)

// Avoid: Using legacy functions without observability
output := pipeline.StartStage(name, workers, fn, input)
```

### 2. Handle Errors Gracefully

```go
// Good: Check errors from router functions
outputs, err := pipeline.RouteByPredicateWithConfig(input, sizes, config, predicates...)
if err != nil {
    logger.Error("Routing failed", pipeline.F("error", err))
    return
}

// Avoid: Using functions that may panic
outputs := pipeline.RouteByPredicate(input, sizes, predicates...)
```

### 3. Use Structured Logging

```go
// Good: Structured fields
logger.Info("Processing started", 
    pipeline.F("stage", "transform"),
    pipeline.F("workers", 4),
    pipeline.F("buffer_size", 100),
)

// Avoid: String interpolation
logger.Info(fmt.Sprintf("Processing started: stage=%s workers=%d", stage, workers))
```

### 4. Combine Observers

```go
// Combine different types of observability
metrics := pipeline.NewMetricsCollector(config)
prometheus := &PrometheusObserver{...}
debugging := &DebugObserver{...}

multiObserver := pipeline.NewMultiObserver(metrics, prometheus)
if debugMode {
    multiObserver.Add(debugging)
}

config.WithObserver(multiObserver)
```

### 5. Use Context for Request Tracking

```go
// Add request/trace IDs to context
ctx := context.WithValue(context.Background(), "trace_id", generateTraceID())
config := pipeline.DefaultConfig().WithContext(ctx)

// Observer can access trace information
observer := pipeline.ObserverFunc(func(ctx context.Context, event pipeline.Event) {
    traceID := ctx.Value("trace_id")
    logger.Info("Pipeline event", 
        pipeline.F("trace_id", traceID),
        pipeline.F("event", event.Type),
    )
})
```

## Backward Compatibility

All existing functions without configuration parameters continue to work:

- `StartStage()` → uses `NoOpObserver` and `NoOpLogger`
- `Fork()` → no observability, but won't panic on errors
- `RouteByPredicate()` → skips unmatched items instead of panicking

For production use, migrate to the `*WithConfig` variants:

- `StartStageWithConfig()`
- `ForkWithConfig()`
- `RouteByPredicateWithConfig()`

## Troubleshooting

### Common Issues

**Q: Events are not being logged**
A: Ensure you're using the `*WithConfig` functions and have set up an observer:

```go
config := pipeline.DefaultConfig().WithObserver(observer)
```

**Q: Pipeline seems slower with observability**
A: Use async observers for expensive operations or filter events by level.

**Q: Getting "unsupported type" warnings**
A: Router functions now handle type mismatches gracefully. Check your type assertions and routing logic.

**Q: Metrics show zero values**
A: Ensure `MetricsCollector` is added to your observer chain and you're using observable functions.

### Debug Mode

Enable debug logging to see all pipeline events:

```go
logger := pipeline.NewConsoleLogger().WithDebug(true)
config := pipeline.DefaultConfig().WithLogger(logger)
```

This will show detailed information about stage lifecycle, worker activity, and item processing.
