package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mimic/internal/core/wrappers"
	"github.com/mimic/internal/fs"
	"github.com/studio-b12/gowebdav"
)

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] <mountpoint> <server>\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	var (
		user       string
		ttl        time.Duration
		maxEntries int
		verbose    bool
		logFile    string
	)

	flag.StringVar(&user, "u", "", "username:password (shorthand)")
	flag.StringVar(&user, "user", "", "username:password")
	flag.DurationVar(&ttl, "ttl", time.Minute, "cache TTL")
	flag.IntVar(&maxEntries, "me", 1000, "cache max entries")
	flag.BoolVar(&verbose, "v", false, "enable verbose logging")
	flag.BoolVar(&verbose, "verbose", false, "enable verbose logging")
	flag.StringVar(&logFile, "l", "", "log file")
	flag.StringVar(&logFile, "log", "", "log file")

	flag.Usage = usage
	flag.Parse()

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

	webdavClient := wrappers.NewWebdavClient(client, ttl, maxEntries)
	filesystem := fs.New(webdavClient)

	if err := filesystem.Mount(mountpoint, []string{}); err != nil {
		fmt.Fprintln(os.Stderr, "Mount failed:", err)
		os.Exit(1)
	}
}
