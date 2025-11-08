package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mimic/internal/core/wrappers"
	"github.com/mimic/internal/fs"
	"github.com/studio-b12/gowebdav"
)

func main() {
	flag.Parse()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered from panic:", r)
			os.Exit(1)
		}
	}()

	client := gowebdav.NewClient("http://localhost:8080", "admin", "password")

	fmt.Println("Trying to connect to the server...")
	if err := client.Connect(); err != nil {
		panic("webdav client: couldn't connect to the server")
	}

	webdavClient := wrappers.NewWebdavClient(client, time.Minute, 1000)

	filesystem := fs.New(webdavClient)
	mountpoint := flag.Arg(0)

	if err := filesystem.Mount(mountpoint, nil); err != nil {
		fmt.Println("Mount failed:", err)
		os.Exit(1)
	}
}
