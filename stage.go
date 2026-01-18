package pipeline

import (
	"fmt"
	"sync"
)

// StartStageWithErr starts a stage with error handling
func StartStageWithErr[I, O any](
	name string,
	workers int,
	fn func(I) (O, error), ec chan error, c ...<-chan I,
) <-chan O {
	// Validate input channels
	if len(c) == 0 {
		ec <- fmt.Errorf("no input channels provided")
		return nil
	}
	var ch <-chan I
	if len(c) == 1 {
		ch = c[0]
	} else {
		ch = Merge(c...)
	}

	// Setup stage workers
	out := make(chan O, workers)
	var stageWg sync.WaitGroup
	stageWg.Add(workers)

	// Initialize stage workers
	for _ = range workers {
		go func() {
			defer stageWg.Done()
			for item := range ch {
				result, err := fn(item)
				if err != nil {
					ec <- fmt.Errorf("%s stage processing error | %w", name, err)
					continue
				}
				out <- result
			}
		}()
	}

	// Close the output channel when stage workers are done
	go func() {
		stageWg.Wait()
		close(out)
	}()

	return out
}

// SafeStartStage starts a stage, returns an error if no input channels are provided
func SafeStartStage[I, O any](
	name string,
	workers int,
	fn func(I) O,
	c ...<-chan I,
) (<-chan O, error) {
	// Validate input channels
	if len(c) == 0 {
		return nil, fmt.Errorf("no input channels provided")
	}
	var ch <-chan I
	if len(c) == 1 {
		ch = c[0]
	} else {
		ch = Merge(c...)
	}

	// Setup stage workers
	out := make(chan O, workers)
	var stageWg sync.WaitGroup
	stageWg.Add(workers)

	// Initialize stage workers
	for _ = range workers {
		go func() {
			defer stageWg.Done()
			for item := range ch {
				result := fn(item)
				out <- result
			}
		}()
	}

	// Close the output channel when stage workers are done
	go func() {
		stageWg.Wait()
		close(out)
	}()

	return out, nil
}

// StartStage starts a stage assuming params are valid
func StartStage[I, O any](
	name string,
	workers int,
	fn func(I) O,
	c ...<-chan I,
) <-chan O {
	var ch <-chan I
	if len(c) == 1 {
		ch = c[0]
	} else {
		ch = Merge(c...)
	}

	// Setup stage workers
	out := make(chan O, workers)
	var stageWg sync.WaitGroup
	stageWg.Add(workers)

	// Initialize stage workers
	for _ = range workers {
		go func() {
			defer stageWg.Done()
			for item := range ch {
				result := fn(item)
				out <- result
			}
		}()
	}

	// Close the output channel when stage workers are done
	go func() {
		stageWg.Wait()
		close(out)
	}()

	return out
}
