package monitor

// Worker events
type WorkerStatusEvent struct {
	Status Status
	URL    string
}

// Process events
type (
	ProcessStoppedEvent struct {
		ServiceID string
	}
	ProcessStatusEvent struct {
		ServiceID string
		Status    Status
		URL       string
	}
)
