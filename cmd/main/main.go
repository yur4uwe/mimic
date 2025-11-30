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
	"github.com/studio-b12/gowebdav"
)

func main() {
	cfg, args, err := config.ParseCommandLineArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to parse config/flags:", err)
		os.Exit(2)
	}

	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "Error: missing required positional arguments: <mountpoint> <server>")
		flag.Usage()
		os.Exit(2)
	}

	mountpoint := args[0]
	server := args[1]

	if cfg.Username == "" || cfg.Password == "" {
		fmt.Fprintln(os.Stderr, "Error: missing credentials; provide -u username:password or set in config")
		flag.Usage()
		os.Exit(2)
	}

	if cfg.Verbose {
		fmt.Printf("mount=%q server=%q user=%q ttl=%s maxEntries=%d\n", mountpoint, server, cfg.Username, cfg.TTL, cfg.MaxEntries)
	}

	client := gowebdav.NewClient(server, cfg.Username, cfg.Password)
	fmt.Println("Trying to connect to the server...")
	if err := client.Connect(); err != nil {
		fmt.Fprintln(os.Stderr, "webdav client: couldn't connect to the server:", err)
		os.Exit(1)
	}
	fmt.Println("Server health check successful")

	logger, err := logger.New(cfg.Verbose, cfg.StdLog, cfg.ErrLog)
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to initialize logger:", err)
		os.Exit(1)
	}
	defer logger.Close()
	cache := cache.NewNodeCache(cfg.TTL, cfg.MaxEntries)

	webdavClient := wrappers.NewWebdavClient(client, cache, server, cfg.Username, cfg.Password, true)
	filesystem := fs.New(webdavClient, logger)

	defer filesystem.Unmount()
	if err := filesystem.Mount(mountpoint, []string{}); err != nil {
		logger.Errorf("Mount failed: %v", err)
		os.Exit(1)
	}
}
