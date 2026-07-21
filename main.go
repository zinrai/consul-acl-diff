package main

import (
	"flag"
	"fmt"
	"os"
)

// Injected at build time by goreleaser via -ldflags -X.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	drift, err := run()
	if err != nil {
		fmt.Fprintln(os.Stderr, "consul-acl-diff:", err)
		os.Exit(2)
	}
	if drift {
		os.Exit(1)
	}
}

// run reports drift and returns whether any was found. Exit codes: 0 when the
// config matches Consul, 1 when drift is found, 2 when an error occurs (handled
// by main).
func run() (bool, error) {
	var (
		configPath  string
		consulAddr  string
		showVersion bool
	)
	flag.StringVar(&configPath, "config", "", "path to configuration file (required)")
	flag.StringVar(&consulAddr, "consul-addr", "http://127.0.0.1:8500", "Consul HTTP API address")
	flag.BoolVar(&showVersion, "version", false, "print version and exit")
	flag.Parse()

	if showVersion {
		fmt.Printf("consul-acl-diff %s (commit %s, built %s)\n", version, commit, date)
		return false, nil
	}

	if configPath == "" {
		return false, fmt.Errorf("-config is required")
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		return false, err
	}

	client := NewConsulClient(consulAddr, os.Getenv("CONSUL_HTTP_TOKEN"))
	diff, err := CalculateDiff(client, cfg)
	if err != nil {
		return false, err
	}

	PrintDiff(diff)
	return diff.HasDrift(), nil
}
