package main

import (
	"fmt"
	"os"
	"time"

	"github.com/kuetix/std-cli/modules"

	"github.com/kuetix/engine"
	"github.com/kuetix/engine/boot"
	"github.com/kuetix/engine/pkg/domain"
)

var Version string
var BuildTime string

func main() {
	if Version == "" {
		Version = "dev"
	}

	if BuildTime == "" {
		BuildTime = time.Now().Format(time.RFC3339)
	}

	modules.Enable()

	// Run the API server startup workflow using engine.RunWorkflow
	options := &boot.Options{
		EngineName:    "cli-pkg",
		ConfigName:    "engine",
		Verbose:       true,
		Quiet:         false,
		Amount:        1,
		Retry:         1,
		RetryDelay:    0,
		RestartPolicy: "",
		Workflow:      "@cli/startup",
		Version:       Version,
		BuildTime:     BuildTime,
		LogPath:       "stdout",
		Config:        &domain.Config{},
		Args: []string{
			"Version: " + Version,
			"BuildTime: " + BuildTime,
		},
		Context: map[string]interface{}{},
	}
	response := engine.RunWorkflow(options)

	for _, res := range response {
		if res.Error != nil {
			fmt.Printf("Error: %s\n", res.Error)
			os.Exit(1)
		}
		if res.Response != nil {
			fmt.Printf("Result: %v\n", res.Response)
		}
	}

	engine.ShutdownEngine()
}
