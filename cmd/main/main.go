package main

import (
	"fmt"
	"os"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/config"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/core/wrappers"
	"github.com/mimic/internal/fs"
	flag "github.com/spf13/pflag"
)

func main() {
	cfg, err := config.ParseCommandLineArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse config/flags:", err)
		os.Exit(2)
	}

	if cfg.Username == "" || cfg.Password == "" {
		fmt.Fprintln(os.Stderr, "Error: missing credentials; provide -u username:password or set in config")
		flag.Usage()
		os.Exit(2)
	}

	if cfg.Verbose {
		fmt.Printf("mount=%q server=%q user=%q ttl=%s maxEntries=%d\n", cfg.Mountpoint, cfg.URL, cfg.Username, cfg.TTL, cfg.MaxEntries)
	}

	logger, err := logger.New(cfg.Verbose, cfg.StdLog, cfg.ErrLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize logger:", err)
		os.Exit(1)
	}
	defer logger.Close()
	cache := cache.NewNodeCache(cfg.TTL, cfg.MaxEntries)

	webdavClient := wrappers.NewWebdavClient(cache, cfg.URL, cfg.Username, cfg.Password, true)
	filesystem := fs.New(webdavClient, logger)

	defer filesystem.Unmount()
	if err := filesystem.Mount(cfg.Mountpoint, []string{}); err != nil {
		logger.Errorf("Mount failed: %v", err)
		os.Exit(1)
	}
}
