package pipeline

// EventType represents different types of pipeline events
type EventType string

const (
	// Stage events
	EventStageStarted       EventType = "stage_started"
	EventStageFinished      EventType = "stage_finished"
	EventWorkerStarted      EventType = "worker_started"
	EventWorkerFinished     EventType = "worker_finished"
	EventItemProcessStarted EventType = "item_process_started"
	EventItemProcessed      EventType = "item_processed"
	EventItemFailed         EventType = "item_failed"

	// Router events
	EventRouteStarted    EventType = "route_started"
	EventRouteFinished   EventType = "route_finished"
	EventRouteError      EventType = "route_error"
	EventUnsupportedType EventType = "unsupported_type"
	EventNoMatch         EventType = "no_match"
	EventConfigError     EventType = "config_error"

	// General events
	EventError   EventType = "error"
	EventWarning EventType = "warning"
	EventInfo    EventType = "info"
	EventDebug   EventType = "debug"
)
