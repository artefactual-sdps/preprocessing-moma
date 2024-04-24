package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/pflag"
	"go.artefactual.dev/tools/log"

	"github.com/artefactual-sdps/preprocessing-moma/cmd/worker/workercmd"
	"github.com/artefactual-sdps/preprocessing-moma/internal/config"
	"github.com/artefactual-sdps/preprocessing-moma/internal/version"
)

func main() {
	p := pflag.NewFlagSet(workercmd.Name, pflag.ExitOnError)
	p.String("config", "", "Configuration file")
	if err := p.Parse(os.Args[1:]); err == flag.ErrHelp {
		os.Exit(1)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var cfg config.Configuration
	configFile, _ := p.GetString("config")
	configFileFound, configFileUsed, err := config.Read(&cfg, configFile)
	if err != nil {
		fmt.Printf("Failed to read configuration: %v\n", err)
		os.Exit(1)
	}

	logger := log.New(os.Stderr,
		log.WithName(workercmd.Name),
		log.WithDebug(cfg.Debug),
		log.WithLevel(cfg.Verbosity),
	)
	defer log.Sync(logger)

	keys := []interface{}{
		"version", version.Long,
		"pid", os.Getpid(),
		"go", runtime.Version(),
	}
	if version.GitCommit != "" {
		keys = append(keys, "commit", version.GitCommit)
	}
	logger.Info("Starting...", keys...)

	if configFileFound {
		logger.Info("Configuration file loaded.", "path", configFileUsed)
	} else {
		logger.Info("Configuration file not found.")
	}

	m := workercmd.NewMain(logger, cfg)

	if err := m.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
