package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mimic/internal/core/webdav"
	"github.com/mimic/internal/fs"
)

func main() {
	flag.Parse()

	client := webdav.NewClient("http://localhost:8080", "admin", "password")

	filesystem := fs.New(client)
	mountpoint := flag.Arg(0)

	// Create a new WebDAV client

	if err := filesystem.Mount(mountpoint); err != nil {
		fmt.Println("Mount failed:", err)
		os.Exit(1)
	}
}
