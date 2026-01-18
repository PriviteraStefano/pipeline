# Pipeline Router Documentation

This document describes the various routing functions available in the pipeline package for distributing data across multiple channels.

## Overview

The router provides several strategies for splitting an input channel into multiple output channels based on different criteria:

- **Type-based routing**: Route items based on their data types
- **Predicate-based routing**: Route items using custom logic functions
- **Key-based routing**: Route items based on extracted keys
- **Load balancing**: Distribute items evenly across workers
- **Broadcasting**: Send items to all output channels
- **Round-robin**: Simple sequential distribution

## Function Reference

### Binary Routing (Original Functions)

#### `ForkUnbuffered[T, U any](c <-chan interface{}) (left chan T, right chan U)`

Routes items from an input channel to two unbuffered output channels based on type.

```go
input := make(chan interface{}, 10)
strings, ints := ForkUnbuffered[string, int](input)

input <- "hello"
input <- 42
input <- "world"
close(input)
```

#### `Fork[T, U any](c <-chan interface{}, leftBufSize, rightBufSize int) (left chan T, right chan U)`

Similar to `ForkUnbuffered` but with configurable buffer sizes.

```go
input := make(chan interface{}, 10)
strings, ints := Fork[string, int](input, 5, 5)
```

### Multi-Way Routing

#### `RouteByPredicate[T any](input <-chan T, bufferSizes []int, predicates ...func(T) bool) []chan T`

Routes items to different channels based on predicate functions. Each item goes to the first matching predicate.

```go
input := make(chan int, 10)

even := func(n int) bool { return n%2 == 0 }
odd := func(n int) bool { return n%2 != 0 }

channels := RouteByPredicate(input, []int{5, 5}, even, odd)
```

**Use cases:**
- Priority-based routing
- Content filtering
- Business rule routing

#### `RouteByKey[T any, K comparable](input <-chan T, bufferSize int, keyFunc func(T) K, keys ...K) map[K]chan T`

Routes items to channels based on a key extraction function.

```go
type Order struct {
    ID       int
    Category string
}

input := make(chan Order, 10)
keyFunc := func(o Order) string { return o.Category }
channels := RouteByKey(input, 5, keyFunc, "electronics", "books", "clothing")

// Access channels by key
electronicsOrders := channels["electronics"]
```

**Use cases:**
- Partitioning by category
- Tenant-based routing
- Geographic distribution

#### `RoundRobin[T any](input <-chan T, numChannels int, bufferSize int) []chan T`

Distributes items sequentially across multiple channels in round-robin fashion.

```go
input := make(chan Task, 100)
workers := RoundRobin(input, 4, 10) // 4 worker channels

for i, worker := range workers {
    go func(id int, ch chan Task) {
        for task := range ch {
            // Process task in worker
        }
    }(i, worker)
}
```

**Use cases:**
- Load distribution
- Worker pool management
- Simple load balancing

#### `Broadcast[T any](input <-chan T, numChannels int, bufferSize int) []chan T`

Sends each item to all output channels simultaneously.

```go
input := make(chan Event, 10)
services := Broadcast(input, 3, 5) // 3 service channels

// Each service gets all events
logger := services[0]
analytics := services[1] 
notifier := services[2]
```

**Use cases:**
- Event distribution
- Service replication
- Monitoring and logging

#### `LoadBalance[T any](input <-chan T, numChannels int, bufferSize int) []chan T`

Routes items to the channel with the smallest buffer, providing dynamic load balancing.

```go
input := make(chan Job, 100)
workers := LoadBalance(input, 4, 10)

// Workers with different processing speeds
for i, worker := range workers {
    go func(id int, ch chan Job) {
        for job := range ch {
            // Processing time varies by worker
            time.Sleep(time.Duration(id+1) * 100 * time.Millisecond)
            processJob(job)
        }
    }(i, worker)
}
```

**Use cases:**
- Dynamic load balancing
- Heterogeneous worker pools
- Adaptive scaling

#### `MultiTypeRoute(input <-chan interface{}, bufferSize int, types ...reflect.Type) map[reflect.Type]chan interface{}`

Routes items based on their runtime types using reflection.

```go
input := make(chan interface{}, 10)

orderType := reflect.TypeOf(Order{})
userType := reflect.TypeOf(User{})

channels := MultiTypeRoute(input, 5, orderType, userType)

orderChan := channels[orderType]
userChan := channels[userType]
```

**Use cases:**
- Heterogeneous data processing
- Dynamic type routing
- Protocol message routing

#### `ConditionalRoute[T any](input <-chan T, condition func(T) bool, matchBufSize, nomatchBufSize int) (match, nomatch chan T)`

Simple binary routing based on a condition function.

```go
input := make(chan Order, 10)

condition := func(o Order) bool { return o.Amount >= 1000.0 }
highValue, regular := ConditionalRoute(input, condition, 5, 5)
```

**Use cases:**
- Binary classification
- Threshold-based routing
- Feature flags

## Performance Considerations

### Buffer Sizing

- **Small buffers (1-10)**: Lower memory usage, potential blocking
- **Medium buffers (10-100)**: Good balance for most use cases
- **Large buffers (100+)**: Higher throughput, more memory usage

### Routing Strategy Selection

| Strategy | Best For | Performance | Memory |
|----------|----------|-------------|---------|
| RoundRobin | Equal load distribution | High | Low |
| LoadBalance | Uneven processing speeds | Medium | Medium |
| RouteByKey | Partitioned processing | High | Medium |
| RouteByPredicate | Complex routing logic | Medium | Low |
| Broadcast | Event distribution | Low | High |

### Goroutine Management

All routing functions create background goroutines. Ensure proper cleanup:

```go
// Always close input channels to terminate routing goroutines
defer close(input)

// Drain output channels if needed
go func() {
    for range outputChannel {
        // Consume remaining items
    }
}()
```

## Error Handling

### Panic Conditions

- **RouteByPredicate**: No predicate matches an item
- **RouteByKey**: No channel exists for extracted key
- **MultiTypeRoute**: Unsupported type encountered
- **Fork/ForkUnbuffered**: Unsupported type for type parameters

### Best Practices

1. **Validate inputs**: Ensure predicates cover all cases
2. **Handle unknown types**: Use default cases or error channels
3. **Monitor goroutines**: Use context for cancellation
4. **Test thoroughly**: Include edge cases in tests

## Examples

See `router_examples.go` for comprehensive usage examples of all routing functions.

## Testing

Run tests with:
```bash
go test -v ./pipeline
```

Run benchmarks with:
```bash
go test -bench=. ./pipeline
```

## Advanced Patterns

### Cascading Routers

```go
// First level: Route by priority
input := make(chan Task, 100)
high, normal := ConditionalRoute(input, isHighPriority, 10, 50)

// Second level: Load balance high priority tasks
highWorkers := LoadBalance(high, 3, 5)

// Third level: Round robin normal tasks
normalWorkers := RoundRobin(normal, 8, 10)
```

### Error Channel Pattern

```go
type Result struct {
    Data  interface{}
    Error error
}

input := make(chan Result, 100)
success, failed := ConditionalRoute(input, 
    func(r Result) bool { return r.Error == nil }, 10, 10)
```

### Fan-Out/Fan-In Pattern

```go
// Fan-out: Distribute work
work := make(chan Job, 100)
workers := RoundRobin(work, 4, 10)

// Fan-in: Collect results
results := make(chan Result, 100)
for _, worker := range workers {
    go func(w chan Job) {
        for job := range w {
            result := processJob(job)
            results <- result
        }
    }(worker)
}
```
