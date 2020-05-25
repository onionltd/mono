package main

import (
	baseconfig "github.com/onionltd/mono/pkg/services/config"
	"time"
)

type config struct {
	baseconfig.BaseConfig

	WWWDir                string        `long:"www" description:"WWW resources directory" required:"yes" env:"WWW_PATH"`
	TemplatesDir          string        `long:"templates" description:"Templates directory" required:"yes" env:"TEMPLATES_PATH"`
	OnionTreeDir          string        `long:"oniontree" description:"OnionTree directory" required:"yes" env:"ONIONTREE_PATH"`
	BadgerDBDir           string        `long:"badgerdb" description:"Badger DB directory" required:"yes" env:"BADGERDB_PATH"`
	MonitorConnectionsMax int64         `long:"monitor-connections-max" description:"Maximum parallel connections" default:"255" env:"MONITOR_CONNECTIONS_MAX"`
	MonitorPingTimeout    time.Duration `long:"monitor-ping-timeout" description:"Maximum time before timeout" default:"15s" env:"MONITOR_PING_TIMEOUT"`
	MonitorPingInterval   time.Duration `long:"monitor-ping-interval" description:"Ping in intervals" default:"1m" env:"MONITOR_PING_INTERVAL"`
}
