package monitor

import "time"

type WorkerConfig struct {
	PingInterval time.Duration
	PingTimeout  time.Duration
}

var DefaultWorkerConfig = WorkerConfig{
	PingInterval: 1 * time.Minute,
	PingTimeout:  25 * time.Second,
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
