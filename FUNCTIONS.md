# Stager Functions - In-Depth Documentation

This document provides comprehensive explanations of each function in the stager package, a Go library for building concurrent data processing pipelines.

## Overview

The stager package implements a pipeline pattern with these key characteristics:
- **Type Safety**: Uses Go generics for compile-time type safety
- **Concurrency**: Each stage can run multiple workers concurrently  
- **Composability**: Stages can be chained together easily
- **Error Handling**: Two variants - one with error propagation, one without
- **Fan-in/Fan-out**: Support for merging multiple channels and distributing work

## Functions

### 1. `StartStageWithErr[I, O]` Function

```go
func StartStageWithErr[I any, O any](
    name string,
    workers int,
    fn func(I) (O, error), 
    ec chan error, 
    c ...<-chan I,
) <-chan O
```

**Purpose**: Creates a concurrent processing stage with comprehensive error handling capabilities.

**Generic Parameters**:
- `I any`: Input type - the type of data this stage receives
- `O any`: Output type - the type of data this stage produces

**Parameters**:
- `name string`: Human-readable name for the stage (used in logging and error messages)
- `workers int`: Number of concurrent goroutines to process items in parallel
- `fn func(I) (O, error)`: Processing function that transforms input to output and can return errors
- `ec chan error`: Error channel where processing errors are sent for centralized error handling
- `c ...<-chan I`: Variadic parameter accepting one or more input channels

**Detailed Operation**:

1. **Input Validation**: 
   - Checks if at least one input channel is provided
   - If no channels provided, sends error to error channel and returns nil

2. **Channel Merging**: 
   - If only one input channel provided, uses it directly
   - If multiple input channels provided, uses `Merge()` to combine them into a single channel
   - This allows the stage to accept input from multiple upstream stages

3. **Worker Pool Setup**: 
   - Creates a buffered output channel with capacity equal to number of workers
   - Initializes a `sync.WaitGroup` to track worker completion
   - Adds all workers to the WaitGroup before starting them

4. **Worker Logic**: Each worker goroutine:
   - Continuously reads items from the input channel using `range` (blocks until channel closed)
   - Logs which worker is processing which stage for debugging
   - Calls the processing function `fn(item)` 
   - If an error occurs, wraps it with stage context and sends to error channel
   - Uses `continue` to skip failed items rather than stopping processing
   - If successful, sends the result to the output channel

5. **Cleanup Management**: 
   - Separate goroutine waits for all workers to finish using `WaitGroup`
   - Closes the output channel when all workers complete
   - Ensures proper resource cleanup and signal to downstream stages

**Error Handling Strategy**:
- Errors don't stop the pipeline - failed items are skipped
- All errors are sent to centralized error channel with stage context
- Downstream stages continue receiving successful results

**Use Cases**:
- Network operations that may fail (HTTP requests, database queries)
- File processing where some files might be corrupted
- Data validation where some records might be invalid
- Any transformation that can encounter recoverable errors

**Example**:
```go
errorChan := make(chan error, 100)
processedData := StartStageWithErr(
    "data-validator", 
    4, 
    validateRecord, 
    errorChan, 
    inputChannel,
)

// Handle errors separately
go func() {
    for err := range errorChan {
        log.Printf("Validation error: %v", err)
    }
}()
```

### 2. `StartStage[I, O]` Function

```go
func StartStage[I any, O any](
    name string,
    workers int,
    fn func(I) O, 
    c ...<-chan I,
) (<-chan O, error)
```

**Purpose**: Creates a concurrent processing stage for operations that are guaranteed to succeed (no error handling).

**Key Differences from `StartStageWithErr`**:
- Processing function `fn func(I) O` doesn't return an error
- No error channel parameter needed
- Returns an error directly if input validation fails
- Simpler, more performant worker logic since there's no error handling overhead

**Parameters**:
- `name string`: Stage identifier for logging
- `workers int`: Number of concurrent worker goroutines
- `fn func(I) O`: Pure transformation function (no errors expected)
- `c ...<-chan I`: One or more input channels

**Detailed Operation**:

1. **Input Validation**: 
   - Returns `(nil, error)` immediately if no input channels provided
   - Fails fast rather than using error channels

2. **Channel Handling**: 
   - Same merging logic as `StartStageWithErr`
   - Supports multiple input channels seamlessly

3. **Worker Pool Setup**: 
   - Identical to error version but simpler worker logic
   - Buffered output channel for better performance

4. **Worker Logic**: Each worker:
   - Reads from input channel until closed
   - Logs processing start for debugging
   - Calls processing function (no error checking needed)
   - Directly sends result to output channel
   - No error handling overhead for maximum performance

5. **Return Values**: 
   - Returns output channel and `nil` error on successful setup
   - Returns `nil` channel and error if setup fails

**Performance Characteristics**:
- Faster than `StartStageWithErr` due to no error handling
- Lower memory overhead (no error channel)
- Simpler goroutine logic reduces context switching

**Use Cases**:
- Mathematical computations that can't fail
- Data formatting and string transformations
- Type conversions between compatible types
- Simple mappings and lookups with guaranteed data
- Performance-critical transformations

**Example**:
```go
// Transform strings to uppercase - can't fail
upperCaseStage, err := StartStage(
    "uppercase", 
    2, 
    strings.ToUpper, 
    stringChannel,
)
if err != nil {
    log.Fatal("Failed to start stage:", err)
}
```

### 3. `Merge[T]` Function

```go
func Merge[T](cs ...<-chan T) <-chan T
```

**Purpose**: Implements a fan-in operation that combines multiple input channels into a single output channel, enabling pipeline convergence.

**Generic Parameter**:
- `T`: The type of data flowing through all channels (must be consistent)

**Detailed Operation**:

1. **Output Channel Creation**: 
   - Creates an unbuffered output channel
   - Unbuffered to maintain backpressure characteristics

2. **WaitGroup Coordination**: 
   - Initializes `sync.WaitGroup` with count equal to number of input channels
   - Ensures all input channels are fully consumed before closing output

3. **Reader Goroutines**: 
   - Spawns one goroutine per input channel
   - Each goroutine:
     - Reads all values from its assigned input channel using `range`
     - Forwards each value immediately to the output channel
     - Calls `wg.Done()` when its input channel closes
     - Runs independently, so slow channels don't block fast ones

4. **Cleanup Coordination**: 
   - Separate goroutine waits for all readers to finish
   - Closes the output channel only after all input channels are exhausted
   - Prevents premature closure that would lose data

**Key Features**:

- **Non-blocking Behavior**: Each input channel is handled by its own goroutine
- **Order Preservation**: Values from each individual channel maintain their order
- **Interleaving**: Values from different channels can interleave non-deterministically
- **Backpressure Propagation**: Unbuffered channel maintains flow control
- **Resource Safety**: Guarantees output channel closure for proper cleanup

**Concurrency Characteristics**:
- **Parallel Reading**: All input channels are read simultaneously
- **Sequential Output**: Only one value can be sent to output at a time
- **Fair Scheduling**: Go's scheduler ensures fair access among reader goroutines

**Use Cases**:

1. **Pipeline Convergence**: Combining results from parallel processing stages
2. **Load Balancing**: Merging outputs from multiple worker pools
3. **Data Aggregation**: Collecting results from distributed computations
4. **Fan-in Patterns**: Converting multiple data streams into single stream

**Performance Considerations**:
- Memory usage scales with number of input channels (one goroutine each)
- Output rate limited by slowest upstream producer
- CPU overhead minimal (just goroutine context switching)

**Example Usage Patterns**:

```go
// Merge results from parallel processors
processor1 := StartStage("proc1", 2, process, input1)
processor2 := StartStage("proc2", 2, process, input2)
processor3 := StartStage("proc3", 2, process, input3)

// Combine all results
combined := Merge(processor1, processor2, processor3)

// Send to final stage
finalResults := StartStage("final", 1, finalize, combined)
```

## Architecture Patterns

### Pipeline Chaining
```go
// Stage 1: Read data (no errors expected)
stage1, _ := StartStage("reader", 2, readFunction, inputChannel)

// Stage 2: Process with potential errors  
errorChan := make(chan error, 100)
stage2 := StartStageWithErr("processor", 4, processFunction, errorChan, stage1)

// Stage 3: Final output
stage3, _ := StartStage("writer", 1, writeFunction, stage2)
```

### Fan-out/Fan-in Pattern
```go
// Fan-out: Split work across multiple stages
stage1a := StartStage("process-a", 2, processA, input)
stage1b := StartStage("process-b", 2, processB, input)

// Fan-in: Merge results back together
merged := Merge(stage1a, stage1b)
final := StartStage("combine", 1, combineResults, merged)
```

### Error Handling Strategy
```go
errorChan := make(chan error, 1000)

// Multiple stages can share the same error channel
stage1 := StartStageWithErr("stage1", 4, fn1, errorChan, input1)
stage2 := StartStageWithErr("stage2", 2, fn2, errorChan, stage1)

// Centralized error handling
go func() {
    for err := range errorChan {
        log.Printf("Pipeline error: %v", err)
        // Could implement retry logic, alerting, etc.
    }
}()
```

## Best Practices

1. **Worker Count Tuning**: Start with `runtime.NumCPU()` and adjust based on workload
2. **Error Channel Sizing**: Buffer error channels to prevent blocking on error bursts
3. **Resource Cleanup**: Always consume error channels to prevent goroutine leaks
4. **Type Safety**: Leverage generics for compile-time type checking
5. **Monitoring**: Use the stage names for logging and metrics collection
6. **Graceful Shutdown**: Close input channels to signal pipeline completion

This architecture enables building robust, scalable data processing pipelines with fine-grained control over concurrency, error handling, and resource management.