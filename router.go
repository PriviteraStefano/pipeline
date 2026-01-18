package pipeline

import (
	"fmt"
	"reflect"
)

// ForkUnbuffered creates two unbuffered channels from a single input channel based on type assertion
func ForkUnbuffered[T, U any](c <-chan any) (left chan T, right chan U) {
	return ForkUnbufferedWithConfig[T, U](c, nil)
}

// ForkUnbufferedWithConfig creates two unbuffered channels with observability
func ForkUnbufferedWithConfig[T, U any](c <-chan any, config *PipelineConfig) (left chan T, right chan U) {
	if config == nil {
		config = DefaultConfig()
	}

	left = make(chan T)
	right = make(chan U)

	EmitEvent(config, EventRouteStarted, "fork_unbuffered", "Starting fork operation", nil, nil)

	go func() {
		defer close(left)
		defer close(right)
		defer EmitEvent(config, EventRouteFinished, "fork_unbuffered", "Fork operation completed", nil, nil)

		for v := range c {
			switch v := v.(type) {
			case T:
				left <- v
			case U:
				right <- v
			default:
				err := fmt.Errorf("unsupported type: %T", v)
				EmitEvent(config, EventUnsupportedType, "fork_unbuffered", err.Error(), err, map[string]any{
					"type":  fmt.Sprintf("%T", v),
					"value": v,
				})
				// Skip the item instead of panicking
				continue
			}
		}
	}()
	return left, right
}

// Fork creates two buffered channels from a single input channel based on type assertion
func Fork[T, U any](c <-chan any, leftBufSize, rightBufSize int) (left chan T, right chan U) {
	return ForkWithConfig[T, U](c, leftBufSize, rightBufSize, nil)
}

// ForkWithConfig creates two buffered channels with observability
func ForkWithConfig[T, U any](c <-chan any, leftBufSize, rightBufSize int, config *PipelineConfig) (left chan T, right chan U) {
	if config == nil {
		config = DefaultConfig()
	}

	left = make(chan T, leftBufSize)
	right = make(chan U, rightBufSize)

	EmitEvent(config, EventRouteStarted, "fork", "Starting fork operation", nil, map[string]any{
		"left_buffer_size":  leftBufSize,
		"right_buffer_size": rightBufSize,
	})

	go func() {
		defer close(left)
		defer close(right)
		defer EmitEvent(config, EventRouteFinished, "fork", "Fork operation completed", nil, nil)

		for v := range c {
			switch v := v.(type) {
			case T:
				left <- v
			case U:
				right <- v
			default:
				err := fmt.Errorf("unsupported type: %T", v)
				EmitEvent(config, EventUnsupportedType, "fork", err.Error(), err, map[string]any{
					"type":  fmt.Sprintf("%T", v),
					"value": v,
				})
				// Skip the item instead of panicking
				continue
			}
		}
	}()
	return left, right
}

// RouteByPredicate routes items to different channels based on predicate functions
// Each predicate function determines if an item should go to the corresponding channel
func RouteByPredicate[T any](input <-chan T, bufferSizes []int, predicates ...func(T) bool) []chan T {
	outputs, _ := RouteByPredicateWithConfig(input, bufferSizes, nil, predicates...)
	return outputs
}

// RouteByPredicateWithConfig routes items with observability and error handling
func RouteByPredicateWithConfig[T any](input <-chan T, bufferSizes []int, config *PipelineConfig, predicates ...func(T) bool) ([]chan T, error) {
	if config == nil {
		config = DefaultConfig()
	}

	if len(predicates) != len(bufferSizes) {
		err := fmt.Errorf("number of predicates (%d) must match number of buffer sizes (%d)", len(predicates), len(bufferSizes))
		EmitEvent(config, EventConfigError, "route_by_predicate", err.Error(), err, nil)
		return nil, err
	}

	outputs := make([]chan T, len(predicates))
	for i, bufSize := range bufferSizes {
		outputs[i] = make(chan T, bufSize)
	}

	EmitEvent(config, EventRouteStarted, "route_by_predicate", "Starting predicate-based routing", nil, map[string]any{
		"predicate_count": len(predicates),
		"buffer_sizes":    bufferSizes,
	})

	go func() {
		defer func() {
			for _, ch := range outputs {
				close(ch)
			}
		}()
		defer EmitEvent(config, EventRouteFinished, "route_by_predicate", "Predicate routing completed", nil, nil)

		for item := range input {
			routed := false
			for i, predicate := range predicates {
				if predicate(item) {
					outputs[i] <- item
					routed = true
					break // Route to first matching predicate only
				}
			}
			if !routed {
				EmitEvent(config, EventNoMatch, "route_by_predicate", "No predicate matched for item", nil, map[string]any{
					"item": fmt.Sprintf("%+v", item),
				})
				// Skip the item instead of panicking
				continue
			}
		}
	}()

	return outputs, nil
}

// RouteByKey routes items to channels based on a key extraction function
// The key function extracts a routing key from each item
func RouteByKey[T any, K comparable](input <-chan T, bufferSize int, keyFunc func(T) K, keys ...K) map[K]chan T {
	outputs, _ := RouteByKeyWithConfig(input, bufferSize, keyFunc, nil, keys...)
	return outputs
}

// RouteByKeyWithConfig routes items by key with observability and error handling
func RouteByKeyWithConfig[T any, K comparable](input <-chan T, bufferSize int, keyFunc func(T) K, config *PipelineConfig, keys ...K) (map[K]chan T, error) {
	if config == nil {
		config = DefaultConfig()
	}

	outputs := make(map[K]chan T)
	for _, key := range keys {
		outputs[key] = make(chan T, bufferSize)
	}

	EmitEvent(config, EventRouteStarted, "route_by_key", "Starting key-based routing", nil, map[string]any{
		"key_count":   len(keys),
		"buffer_size": bufferSize,
	})

	go func() {
		defer func() {
			for _, ch := range outputs {
				close(ch)
			}
		}()
		defer EmitEvent(config, EventRouteFinished, "route_by_key", "Key routing completed", nil, nil)

		for item := range input {
			key := keyFunc(item)
			if ch, exists := outputs[key]; exists {
				ch <- item
			} else {
				EmitEvent(config, EventNoMatch, "route_by_key", "No output channel for key", nil, map[string]any{
					"key":  key,
					"item": fmt.Sprintf("%+v", item),
				})
				// Skip the item instead of panicking
				continue
			}
		}
	}()

	return outputs, nil
}

// MultiTypeRoute routes items to different channels based on their types
// Uses reflection to determine the appropriate channel for each type
func MultiTypeRoute(input <-chan any, bufferSize int, types ...reflect.Type) map[reflect.Type]chan any {
	outputs, _ := MultiTypeRouteWithConfig(input, bufferSize, nil, types...)
	return outputs
}

// MultiTypeRouteWithConfig routes items by type with observability and error handling
func MultiTypeRouteWithConfig(input <-chan any, bufferSize int, config *PipelineConfig, types ...reflect.Type) (map[reflect.Type]chan any, error) {
	if config == nil {
		config = DefaultConfig()
	}

	outputs := make(map[reflect.Type]chan any)
	for _, t := range types {
		outputs[t] = make(chan any, bufferSize)
	}

	EmitEvent(config, EventRouteStarted, "multi_type_route", "Starting type-based routing", nil, map[string]any{
		"type_count":  len(types),
		"buffer_size": bufferSize,
	})

	go func() {
		defer func() {
			for _, ch := range outputs {
				close(ch)
			}
		}()
		defer EmitEvent(config, EventRouteFinished, "multi_type_route", "Type routing completed", nil, nil)

		for item := range input {
			itemType := reflect.TypeOf(item)
			if ch, exists := outputs[itemType]; exists {
				ch <- item
			} else {
				EmitEvent(config, EventUnsupportedType, "multi_type_route", "Unsupported type encountered", nil, map[string]any{
					"type": itemType.String(),
					"item": fmt.Sprintf("%+v", item),
				})
				// Skip the item instead of panicking
				continue
			}
		}
	}()

	return outputs, nil
}

// ConditionalRoute routes items based on a condition function
// Items matching the condition go to the 'match' channel, others to 'nomatch'
func ConditionalRoute[T any](input <-chan T, condition func(T) bool, matchBufSize, nomatchBufSize int) (match, nomatch chan T) {
	return ConditionalRouteWithConfig(input, condition, matchBufSize, nomatchBufSize, nil)
}

// ConditionalRouteWithConfig routes items conditionally with observability
func ConditionalRouteWithConfig[T any](input <-chan T, condition func(T) bool, matchBufSize, nomatchBufSize int, config *PipelineConfig) (match, nomatch chan T) {
	if config == nil {
		config = DefaultConfig()
	}

	match = make(chan T, matchBufSize)
	nomatch = make(chan T, nomatchBufSize)

	EmitEvent(config, EventRouteStarted, "conditional_route", "Starting conditional routing", nil, map[string]any{
		"match_buffer_size":   matchBufSize,
		"nomatch_buffer_size": nomatchBufSize,
	})

	go func() {
		defer close(match)
		defer close(nomatch)
		defer EmitEvent(config, EventRouteFinished, "conditional_route", "Conditional routing completed", nil, nil)

		for item := range input {
			if condition(item) {
				match <- item
			} else {
				nomatch <- item
			}
		}
	}()

	return match, nomatch
}
