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
	return StartStageWithErrAndConfig(name, workers, fn, ec, nil, c...)
}

// StartStageWithErrAndConfig starts a stage with error handling and observability
func StartStageWithErrAndConfig[I, O any](
	name string,
	workers int,
	fn func(I) (O, error),
	ec chan error,
	config *PipelineConfig,
	c ...<-chan I,
) <-chan O {
	if config == nil {
		config = DefaultConfig()
	}
	// Validate input channels
	if len(c) == 0 {
		err := fmt.Errorf("no input channels provided for stage %s", name)
		ec <- err
		EmitStageEvent(config, EventError, name, 0, "Stage initialization failed", err)
		return nil
	}
	var ch <-chan I
	if len(c) == 1 {
		ch = c[0]
	} else {
		ch = Merge(c...)
	}

	EmitStageEvent(config, EventStageStarted, name, 0, fmt.Sprintf("Starting stage with %d workers", workers), nil)

	// Setup stage workers
	out := make(chan O, workers)
	var stageWg sync.WaitGroup
	stageWg.Add(workers)

	// Initialize stage workers
	for workerID := range workers {
		go func(id int) {
			defer stageWg.Done()
			EmitStageEvent(config, EventWorkerStarted, name, id, "Worker started", nil)
			defer EmitStageEvent(config, EventWorkerFinished, name, id, "Worker finished", nil)

			for item := range ch {
				result, err := fn(item)
				if err != nil {
					wrappedErr := fmt.Errorf("%s stage processing error | %w", name, err)
					ec <- wrappedErr
					EmitStageEvent(config, EventItemFailed, name, id, "Item processing failed", wrappedErr)
					continue
				}
				out <- result
				EmitStageEvent(config, EventItemProcessed, name, id, "Item processed successfully", nil)
			}
		}(workerID)
	}

	// Close the output channel when stage workers are done
	go func() {
		stageWg.Wait()
		close(out)
		EmitStageEvent(config, EventStageFinished, name, 0, "Stage completed", nil)
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
	return SafeStartStageWithConfig(name, workers, fn, nil, c...)
}

// SafeStartStageWithConfig starts a stage with observability, returns an error if no input channels are provided
func SafeStartStageWithConfig[I, O any](
	name string,
	workers int,
	fn func(I) O,
	config *PipelineConfig,
	c ...<-chan I,
) (<-chan O, error) {
	if config == nil {
		config = DefaultConfig()
	}
	// Validate input channels
	if len(c) == 0 {
		err := fmt.Errorf("no input channels provided for stage %s", name)
		EmitStageEvent(config, EventError, name, 0, "Stage initialization failed", err)
		return nil, err
	}
	var ch <-chan I
	if len(c) == 1 {
		ch = c[0]
	} else {
		ch = Merge(c...)
	}

	EmitStageEvent(config, EventStageStarted, name, 0, fmt.Sprintf("Starting stage with %d workers", workers), nil)

	// Setup stage workers
	out := make(chan O, workers)
	var stageWg sync.WaitGroup
	stageWg.Add(workers)

	// Initialize stage workers
	for workerID := range workers {
		go func(id int) {
			defer stageWg.Done()
			EmitStageEvent(config, EventWorkerStarted, name, id, "Worker started", nil)
			defer EmitStageEvent(config, EventWorkerFinished, name, id, "Worker finished", nil)

			for item := range ch {
				result := fn(item)
				out <- result
				EmitStageEvent(config, EventItemProcessed, name, id, "Item processed successfully", nil)
			}
		}(workerID)
	}

	// Close the output channel when stage workers are done
	go func() {
		stageWg.Wait()
		close(out)
		EmitStageEvent(config, EventStageFinished, name, 0, "Stage completed", nil)
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
	return StartStageWithConfig(name, workers, fn, nil, c...)
}

// StartStageWithConfig starts a stage with observability assuming params are valid
func StartStageWithConfig[I, O any](
	name string,
	workers int,
	fn func(I) O,
	config *PipelineConfig,
	c ...<-chan I,
) <-chan O {
	if config == nil {
		config = DefaultConfig()
	}
	var ch <-chan I
	if len(c) == 1 {
		ch = c[0]
	} else {
		ch = Merge(c...)
	}

	EmitStageEvent(config, EventStageStarted, name, 0, fmt.Sprintf("Starting stage with %d workers", workers), nil)

	// Setup stage workers
	out := make(chan O, workers)
	var stageWg sync.WaitGroup
	stageWg.Add(workers)

	// Initialize stage workers
	for workerID := range workers {
		go func(id int) {
			defer stageWg.Done()
			EmitStageEvent(config, EventWorkerStarted, name, id, "Worker started", nil)
			defer EmitStageEvent(config, EventWorkerFinished, name, id, "Worker finished", nil)

			for item := range ch {
				result := fn(item)
				out <- result
				EmitStageEvent(config, EventItemProcessed, name, id, "Item processed successfully", nil)
			}
		}(workerID)
	}

	// Close the output channel when stage workers are done
	go func() {
		stageWg.Wait()
		close(out)
		EmitStageEvent(config, EventStageFinished, name, 0, "Stage completed", nil)
	}()

	return out
}
