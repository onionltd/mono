package monitor

// Worker events
type workerStatusEvent struct {
	Status Status
	URL    string
}

// Process events
type (
	processStoppedEvent struct {
		ServiceID string
	}
	processStatusEvent struct {
		ServiceID string
		Status    Status
		URL       string
	}
)
