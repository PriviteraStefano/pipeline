# Pipeline

[![Go 1.21+](https://img.shields.io/badge/go-1.21%2B-blue.svg)](https://golang.org/doc/go1.21)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Description

Pipeline is a Go library that provides building blocks for creating concurrent data processing pipelines using channels and goroutines. It offers comprehensive utilities for routing, processing, and managing data flow in high-performance concurrent applications with observability, structured logging and error handling.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Observability Features](#observability-features)
- [Usage](#usage)
- [Features](#features)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Migration Guide](#migration-guide)
- [Contributing](#contributing)
- [License](#license)

## Installation

```bash
go get github.com/PriviteraStefano/pipeline
```

## Quick Start

### Basic Pipeline with Observability

```go
package main

import (
    "fmt"
    "pipeline"
)

func main() {
    // Create a logger
    logger := pipeline.NewConsoleLogger().WithDebug(true)
    config := pipeline.DefaultConfig().WithLogger(logger)
    
    // Create input
    input := make(chan int, 5)
    for i := 1; i <= 5; i++ {
        input <- i
    }
    close(input)
    
    // Process with full observability
    output := pipeline.StartStageWithConfig(
        "multiply", 2,
        func(x int) int { return x * 2 },
        config, input,
    )
    
    for result := range output {
        fmt.Println("Result:", result)
    }
}
```

## Observability Features

### 🔍 Event-Driven Observability
- **Stage Events**: Track stage lifecycle, worker activity, and item processing
- **Router Events**: Monitor routing operations and handle type mismatches gracefully
- **Custom Observers**: Implement your own event handlers for metrics, monitoring, etc.

### 📝 Structured Logging
- **Multiple Logger Types**: Console, JSON, Standard Go logger, and custom implementations
- **Level Filtering**: Debug, Info, Warn, Error levels with configurable filtering
- **Multi-Logger Support**: Write to multiple destinations simultaneously

### 📊 Built-in Metrics
- **Performance Tracking**: Items processed, throughput, stage duration
- **Error Monitoring**: Failed items, routing errors, configuration issues
- **Worker Statistics**: Worker utilization and lifecycle tracking

### 🛡️ Production-Ready Error Handling
- **No More Panics**: Graceful error handling instead of crashing
- **Error Recovery**: Continue processing even when individual items fail
- **Detailed Error Context**: Rich error information with metadata

### Example: Full Observability Setup

```go
// Create comprehensive observability
logger := pipeline.NewConsoleLogger().WithDebug(true)
config := pipeline.DefaultConfig().WithLogger(logger)

// Add metrics collection
metrics := pipeline.NewMetricsCollector(config)
observer := pipeline.NewMultiObserver(
    pipeline.NewLoggerObserver(logger),
    metrics,
)
config.WithObserver(observer)

// Use with any pipeline operation
output := pipeline.StartStageWithConfig("process", 4, processor, config, input)

// Get metrics after processing
finalMetrics := metrics.GetMetrics()
fmt.Printf("Processed %d items in %v\n", 
    finalMetrics.ItemsProcessed, finalMetrics.Duration)
```

## Usage

### Legacy API (Still Supported)

```go
package main

import (
    "fmt"
    "github.com/PriviteraStefano/pipeline"
)

func main() {
    // Create input channel
    input := make(chan int, 10)
    
    // Start a processing stage
    output := pipeline.StartStage("multiply", 3, func(x int) int {
        return x * 2
    }, input)
    
    // Send data
    go func() {
        defer close(input)
        for i := 1; i <= 5; i++ {
            input <- i
        }
    }()
    
    // Receive results
    for result := range output {
        fmt.Println(result)
    }
}
```

### Modern API with Observability

```go
// Create configuration
logger := pipeline.NewJSONLogger(os.Stdout)
config := pipeline.DefaultConfig().WithLogger(logger)

// Process with error handling and observability
errorChan := make(chan error, 10)
output := pipeline.StartStageWithErrAndConfig(
    "parse_data", 3,
    func(data string) (int, error) {
        return strconv.Atoi(data)
    },
    errorChan, config, input,
)
```

### Channel Routing with Error Recovery

```go
// Route with observability and error handling
config := pipeline.DefaultConfig().WithLogger(logger)

// Graceful routing - no panics on unmatched items
outputs, err := pipeline.RouteByPredicateWithConfig(
    input, []int{10, 10}, config,
    func(x int) bool { return x%2 == 0 }, // even numbers
    func(x int) bool { return x%2 == 1 }, // odd numbers
)
if err != nil {
    log.Fatal("Routing setup failed:", err)
}
```

### Error Handling with Result Type

```go
// Create results
success := pipeline.Success[string, string, error]("Operation completed")
warning := pipeline.Warning[string, string, error]("Done", "Minor issue occurred")
err := pipeline.Error[string, string, error]("Failed", fmt.Errorf("critical error"))

// Handle results
pipeline.Manage(success,
    func(s string) { fmt.Println("Success:", s) },
    func(w string) { fmt.Println("Warning:", w) },
    func(e error) { fmt.Println("Error:", e) },
)
```

## Features

### 🔧 Pipeline Processing
- **Multi-worker stages**: Process data concurrently with configurable worker pools
- **Observability**: Complete visibility into stage lifecycle and performance
- **Error handling**: Built-in error propagation with detailed context
- **Safe operations**: Validation and graceful error recovery

### 🔀 Channel Routing
- **Graceful Error Handling**: No panics - unmatched items are logged and skipped
- **Fork**: Split data into two channels with type safety
- **Conditional routing**: Route data based on custom predicates
- **Key-based routing**: Route data using key extraction functions
- **Type-based routing**: Route data based on runtime type information

### 📊 Observability & Monitoring
- **Event System**: Comprehensive event tracking for all operations
- **Structured Logging**: JSON, console, and custom logger support
- **Metrics Collection**: Built-in performance and error metrics
- **Custom Observers**: Integrate with monitoring systems (Prometheus, etc.)

### 🔧 Utilities
- **Channel merging**: Combine multiple input channels into one
- **Generic support**: Full Go generics support for type safety
- **Concurrent processing**: Built on Go's powerful concurrency primitives

### ✅ Result Management
- **Functional error handling**: Result type for success/warning/error states
- **Type-safe operations**: Generic Result type with compile-time safety
- **Flexible handling**: Custom handlers for different result states

## API Reference

### Core Stage Functions

**Legacy API (No Observability)**
- `StartStage[I, O](name, workers, fn, channels...)` - Create a processing stage
- `StartStageWithErr[I, O](name, workers, fn, errorChan, channels...)` - Stage with error handling
- `SafeStartStage[I, O](name, workers, fn, channels...)` - Stage with validation

**Modern API (With Observability)**
- `StartStageWithConfig[I, O](name, workers, fn, config, channels...)` - Stage with full observability
- `StartStageWithErrAndConfig[I, O](name, workers, fn, errorChan, config, channels...)` - Error handling + observability
- `SafeStartStageWithConfig[I, O](name, workers, fn, config, channels...)` - Validation + observability

### Routing Functions

**Legacy API**
- `Fork[T, U](input, leftBuf, rightBuf)` - Split channel by type (may panic)
- `ConditionalRoute[T](input, condition, matchBuf, nomatchBuf)` - Binary routing
- `RouteByKey[T, K](input, bufferSize, keyFunc, keys...)` - Key-based routing (may panic)
- `RouteByPredicate[T](input, bufferSizes, predicates...)` - Predicate routing (may panic)

**Modern API (With Error Handling & Observability)**
- `ForkWithConfig[T, U](input, leftBuf, rightBuf, config)` - Graceful forking
- `ConditionalRouteWithConfig[T](input, condition, matchBuf, nomatchBuf, config)` - Binary routing
- `RouteByKeyWithConfig[T, K](input, bufferSize, keyFunc, config, keys...)` - Key routing with recovery
- `RouteByPredicateWithConfig[T](input, bufferSizes, config, predicates...)` - Predicate routing with recovery

### Observability Functions

- `DefaultConfig()` - Create default configuration
- `NewConsoleLogger()` - Create console logger
- `NewJSONLogger(writer)` - Create JSON logger  
- `NewLoggerObserver(logger)` - Convert logger to observer
- `NewMetricsCollector(config)` - Create metrics collector
- `NewMultiObserver(observers...)` - Combine multiple observers

### Utility Functions

- `Merge[T](channels...)` - Combine multiple channels

## Examples

### Data Processing Pipeline

```go
// Process numbers through multiple stages
numbers := make(chan int, 100)

// Stage 1: Filter even numbers
evenNumbers := pipeline.StartStage("filter-even", 2, func(x int) int {
    if x%2 == 0 {
        return x
    }
    return 0 // or handle differently
}, numbers)

// Stage 2: Square the numbers
squared := pipeline.StartStage("square", 3, func(x int) int {
    return x * x
}, evenNumbers)

// Collect results
for result := range squared {
    if result > 0 {
        fmt.Println("Result:", result)
    }
}
```

### Error Handling with Observability

```go
// Setup observability
logger := pipeline.NewConsoleLogger()
config := pipeline.DefaultConfig().WithLogger(logger)

// Create error channel
errorChan := make(chan error, 10)

// Process with error handling
output := pipeline.StartStageWithErrAndConfig(
    "parse_numbers", 2,
    func(s string) (int, error) {
        return strconv.Atoi(s)
    },
    errorChan, config, input,
)

// Handle errors concurrently
go func() {
    for err := range errorChan {
        fmt.Printf("Processing error: %v\n", err)
    }
}()

// Process results
for result := range output {
    fmt.Printf("Parsed: %d\n", result)
}
```

### Router Error Recovery

```go
// Before: Could panic on type mismatches
outputs := pipeline.Fork[string, int](mixedInput, 10, 10)

// After: Handles errors gracefully with logging
config := pipeline.DefaultConfig().WithLogger(logger)
outputs, err := pipeline.ForkWithConfig[string, int](mixedInput, 10, 10, config)
if err != nil {
    log.Printf("Fork setup error: %v", err)
    return
}
// Unsupported types are logged as warnings and skipped
```

### Custom Integration Example

```go
// Custom observer for Prometheus metrics
type PrometheusObserver struct {
    itemsProcessed *prometheus.CounterVec
}

func (p *PrometheusObserver) OnEvent(ctx context.Context, event pipeline.Event) {
    if event.Type == pipeline.EventItemProcessed {
        p.itemsProcessed.WithLabelValues(event.StageName).Inc()
    }
}

// Use with pipeline
config := pipeline.DefaultConfig().WithObserver(&PrometheusObserver{...})
```

## Migration Guide

### From Legacy API to Modern API

The library maintains full backward compatibility. Legacy functions continue to work but don't provide observability.

**Stage Functions**
```go
// Old (still works, but no observability)
output := pipeline.StartStage("process", 4, processor, input)

// New (recommended for production)
config := pipeline.DefaultConfig().WithLogger(logger)
output := pipeline.StartStageWithConfig("process", 4, processor, config, input)
```

**Router Functions**
```go
// Old (may panic on errors)
outputs := pipeline.RouteByPredicate(input, buffers, predicates...)

// New (graceful error handling)
outputs, err := pipeline.RouteByPredicateWithConfig(input, buffers, config, predicates...)
if err != nil {
    // Handle configuration errors
}
// Runtime errors (unmatched items) are logged and skipped
```

### Observability Migration Steps

1. **Add Basic Logging**
   ```go
   logger := pipeline.NewConsoleLogger()
   config := pipeline.DefaultConfig().WithLogger(logger)
   ```

2. **Replace Function Calls**
   - `StartStage()` → `StartStageWithConfig()`
   - `Fork()` → `ForkWithConfig()`
   - etc.

3. **Add Metrics Collection (Optional)**
   ```go
   metrics := pipeline.NewMetricsCollector(config)
   observer := pipeline.NewMultiObserver(
       pipeline.NewLoggerObserver(logger),
       metrics,
   )
   config.WithObserver(observer)
   ```

4. **Handle Router Errors**
   ```go
   // Check return values from router functions
   outputs, err := pipeline.RouteByPredicateWithConfig(...)
   if err != nil {
       // Handle setup errors
   }
   ```

See `docs/observability.md` for complete documentation and examples.

## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests. 
Make sure to follow Go best practices and include tests for new functionality.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
