package scanner

type Event interface{}

type ScanEvent struct {
	Status    Status
	URL       string
	ServiceID string
	Directory string
	Error     error
}

type (
	WorkerStarted struct {
		URL       string
		ServiceID string
	}

	WorkerStopped struct {
		URL       string
		ServiceID string
		Error     error
	}

	workerStatus struct {
		Status Status
		URL    string
		Error  error
	}
)

type (
	ProcessStarted struct {
		ServiceID string
	}

	ProcessStopped struct {
		ServiceID string
		Error     error
	}

	processStatus struct {
		Status    Status
		URL       string
		ServiceID string
		Error     error
	}
)
