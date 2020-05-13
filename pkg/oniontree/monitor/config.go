package monitor

import "time"

type WorkerConfig struct {
	PingInterval      time.Duration
	PingTimeout       time.Duration
	PingPause         time.Duration
	PingRetryInterval time.Duration
	PingRetryAttempts int
}

var DefaultWorkerConfig = WorkerConfig{
	PingInterval:      1 * time.Minute,
	PingTimeout:       50 * time.Second,
	PingPause:         5 * time.Minute,
	PingRetryInterval: 10 * time.Second,
	PingRetryAttempts: 3,
}

type MonitorConfig struct {
	// WorkerConnectionsMax limits number of parallel outbound TCP connections.
	WorkerTCPConnectionsMax int64
	// WorkerConfig is a configuration passed to workers.
	WorkerConfig WorkerConfig
}

var DefaultMonitorConfig = MonitorConfig{
	WorkerTCPConnectionsMax: 256,
	WorkerConfig:            DefaultWorkerConfig,
}
