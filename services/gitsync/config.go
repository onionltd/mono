package main

import (
	baseconfig "github.com/onionltd/mono/pkg/config"
	"time"
)

type config struct {
	baseconfig.BaseConfig

	OutputDir         string        `long:"output" description:"Output directory" required:"yes" env:"OUTPUT_PATH"`
	Repository        string        `long:"repo" description:"Git repository to sync" required:"yes" env:"REPOSITORY"`
	SyncInterval      time.Duration `long:"sync-interval" description:"Sync Git repository in intervals" default:"2m" env:"REPOSITORY_SYNC_INTERVAL"`
	CloneDepth        int           `long:"sync-depth" description:"Limit sync to the specified number of commits" env:"REPOSITORY_SYNC_DEPTH"`
	NoVerifySignature bool          `long:"no-verify-signature" description:"Verify PGP signed commits" env:"NO_VERIFY_SIGNATURE"`
	KeyRingPath       string        `long:"keyring" description:"Path to armored PGP keyring" env:"PGP_KEYRING"`
}
