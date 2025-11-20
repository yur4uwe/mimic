package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/core/wrappers"
	"github.com/mimic/internal/fs"
	flag "github.com/spf13/pflag"
	"github.com/studio-b12/gowebdav"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <mountpoint> <server>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var (
		userPtr       *string
		ttlPtr        *time.Duration
		maxEntriesPtr *int
		verbosePtr    *bool
		logOutputs    []string
	)

	userPtr = flag.StringP("user", "u", "", "username:password (shorthand)")
	ttlPtr = flag.DurationP("ttl", "t", time.Minute, "cache TTL")
	maxEntriesPtr = flag.IntP("max-entries", "m", 1000, "cache max entries")
	verbosePtr = flag.BoolP("verbose", "v", false, "enable verbose logging")
	logOutputs = *flag.StringSliceP("log", "l", []string{}, "log outputs: stdout, stderr, or txt file paths")

	flag.Usage = usage
	flag.Parse()

	user := *userPtr
	ttl := *ttlPtr
	maxEntries := *maxEntriesPtr
	verbose := *verbosePtr

	if flag.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "Error: missing required positional arguments: <mountpoint> <server>")
		flag.Usage()
		os.Exit(2)
	}

	mountpoint := flag.Arg(0)
	server := flag.Arg(1)

	if user == "" {
		fmt.Fprintln(os.Stderr, "Error: missing credentials; provide -u username:password")
		flag.Usage()
		os.Exit(2)
	}

	parts := strings.SplitN(user, ":", 2)
	username := parts[0]
	password := ""
	if len(parts) > 1 {
		password = parts[1]
	}
	if username == "" || password == "" {
		fmt.Fprintln(os.Stderr, "Error: credentials must be in form username:password")
		os.Exit(2)
	}

	if verbose {
		fmt.Printf("mount=%q server=%q user=%q ttl=%s maxEntries=%d\n", mountpoint, server, username, ttl, maxEntries)
	}

	client := gowebdav.NewClient(server, username, password)
	fmt.Println("Trying to connect to the server...")
	if err := client.Connect(); err != nil {
		fmt.Fprintln(os.Stderr, "webdav client: couldn't connect to the server:", err)
		os.Exit(1)
	}

	logger := logger.New(verbose, logOutputs)
	cache := cache.NewNodeCache(ttl, maxEntries)

	webdavClient := wrappers.NewWebdavClient(client, cache)
	filesystem := fs.New(webdavClient, logger)

	defer filesystem.Unmount()
	if err := filesystem.Mount(mountpoint, []string{}); err != nil {
		logger.Errorf("Mount failed: %v", err)
		os.Exit(1)
	}
}
