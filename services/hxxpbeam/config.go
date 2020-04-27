package main

import "time"

type config struct {
	Listen                string        `long:"listen" description:"Listen on address" default:":8080" env:"HTTP_LISTEN"`
	OnionTreeDir          string        `long:"oniontree" description:"OnionTree directory" required:"yes" env:"ONIONTREE_PATH"`
	MonitorConnectionsMax int64         `long:"monitor-connections-max" description:"Maximum parallel connections" default:"255" env:"MONITOR_CONNECTIONS_MAX"`
	MonitorPingTimeout    time.Duration `long:"monitor-ping-timeout" description:"Maximum time before timeout" default:"15s" env:"MONITOR_PING_TIMEOUT"`
	MonitorPingInterval   time.Duration `long:"monitor-ping-interval" description:"Ping in intervals" default:"1m" env:"MONITOR_PING_INTERVAL"`
	LogLevel              string        `long:"log-level" description:"Set log level" default:"info" env:"LOG_LEVEL"`
}
