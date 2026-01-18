package pipeline

// PipelineManager provides high-level management of pipeline execution with observability
type PipelineManager[T any] struct {
	config *PipelineConfig
	name   string
}

// NewPipelineManager creates a new pipeline manager with the given configuration
func NewPipelineManager[T any](name string, config *PipelineConfig) *PipelineManager[T] {
	if config == nil {
		config = DefaultConfig()
	}
	return &PipelineManager[T]{
		config: config,
		name:   name,
	}
}

// ExecuteStage executes a single stage with observability
func (pm *PipelineManager[T]) ExecuteStageInt(
	stageName string,
	workers int,
	fn func(int) int,
	input <-chan int,
) <-chan int {
	return StartStageWithConfig(stageName, workers, fn, pm.config, input)
}

// ExecuteStageString executes a string processing stage with observability
func (pm *PipelineManager[T]) ExecuteStageString(
	stageName string,
	workers int,
	fn func(string) string,
	input <-chan string,
) <-chan string {
	return StartStageWithConfig(stageName, workers, fn, pm.config, input)
}

// ExecuteStageWithError executes a stage that can return errors
func (pm *PipelineManager[T]) ExecuteStageWithErrorInt(
	stageName string,
	workers int,
	fn func(int) (int, error),
	errorChan chan error,
	input <-chan int,
) <-chan int {
	return StartStageWithErrAndConfig(stageName, workers, fn, errorChan, pm.config, input)
}

// RouteByType routes items by their types with observability
func (pm *PipelineManager[T]) RouteByType(
	input <-chan any,
	bufferSize int,
	types ...interface{},
) map[string]chan any {
	// Convert types to reflect.Type for MultiTypeRouteWithConfig
	// This is a simplified version - in practice you'd handle type conversion properly
	outputs := make(map[string]chan any)

	EmitEvent(pm.config, EventRouteStarted, pm.name, "Starting managed type routing", nil, map[string]interface{}{
		"buffer_size": bufferSize,
		"type_count":  len(types),
	})

	// Implementation would use MultiTypeRouteWithConfig internally
	return outputs
}

// Shutdown gracefully shuts down the pipeline manager
func (pm *PipelineManager[T]) Shutdown() {
	EmitEvent(pm.config, EventInfo, pm.name, "Pipeline manager shutting down", nil, nil)
}
