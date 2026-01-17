# Pipeline

[![Go 1.21+](https://img.shields.io/badge/go-1.21%2B-blue.svg)](https://golang.org/doc/go1.21)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## Description

Pipeline is a Go library that provides building blocks for creating concurrent data processing pipelines using channels and goroutines. It offers a comprehensive set of utilities for routing, processing, and managing data flow in high-performance concurrent applications.

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Features](#features)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Contributing](#contributing)
- [License](#license)

## Installation

```bash
go get github.com/your-username/pipeline
```

## Usage

### Basic Pipeline Stage

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

### Channel Routing

```go
// Route data based on conditions
match, nomatch := pipeline.ConditionalRoute(input, func(x int) bool {
    return x%2 == 0 // route even numbers to 'match'
}, 10, 10)
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

### Pipeline Processing
- **Multi-worker stages**: Process data concurrently with configurable worker pools
- **Error handling**: Built-in error propagation and handling mechanisms
- **Safe operations**: Validation and error checking for robust pipeline construction

### Channel Routing
- **Fork**: Split data into two channels
- **Conditional routing**: Route data based on custom predicates
- **Key-based routing**: Route data using key extraction functions
- **Type-based routing**: Route data based on runtime type information

### Utilities
- **Channel merging**: Combine multiple input channels into one
- **Generic support**: Full Go generics support for type safety
- **Concurrent processing**: Built on Go's powerful concurrency primitives

### Result Management
- **Functional error handling**: Result type for success/warning/error states
- **Type-safe operations**: Generic Result type with compile-time safety
- **Flexible handling**: Custom handlers for different result states

## API Reference

### Core Functions

- `StartStage[I, O](name, workers, fn, channels...)` - Create a processing stage
- `StartStageWithErr[I, O](name, workers, fn, errorChan, channels...)` - Stage with error handling
- `SafeStartStage[I, O](name, workers, fn, channels...)` - Stage with validation

### Routing Functions

- `Fork[T, U](input, leftBuf, rightBuf)` - Split channel by type
- `ConditionalRoute[T](input, condition, matchBuf, nomatchBuf)` - Binary routing
- `RouteByKey[T, K](input, bufferSize, keyFunc, keys...)` - Key-based routing
- `RouteByPredicate[T](input, bufferSizes, predicates...)` - Predicate-based routing

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


## Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests. 
Make sure to follow Go best practices and include tests for new functionality.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
