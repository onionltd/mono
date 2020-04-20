package main

import (
	"time"
)

type config struct {
	OutputDir         string        `long:"output" description:"Output directory" required:"yes" env:"OUTPUT_PATH"`
	Repository        string        `long:"repo" description:"Git repository to sync" required:"yes" env:"REPOSITORY"`
	SyncInterval      time.Duration `long:"sync-interval" description:"Sync Git repository in intervals" default:"2m" env:"REPOSITORY_SYNC_INTERVAL"`
	CloneDepth        int           `long:"depth" description:"Limit fetching to the specified number of commits" env:"GIT_DEPTH"`
	NoVerifySignature bool          `long:"no-verify-signature" description:"Verify PGP signed commits" env:"NO_VERIFY_SIGNATURE"`
	KeyRingPath       string        `long:"keyring" description:"Path to armored PGP keyring" env:"PGP_KEYRING"`
	LogLevel          string        `long:"log-level" description:"Set log level" default:"info" env:"LOG_LEVEL"`
}
