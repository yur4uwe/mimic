package main

import (
	"flag"
	"fmt"
	"os"

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

	if err := client.Connect(); err != nil {
		panic("webdav client: couldn't connect to the server")
	}

	filesystem := fs.New(client)
	mountpoint := flag.Arg(0)

	if err := filesystem.Mount(mountpoint); err != nil {
		fmt.Println("Mount failed:", err)
		os.Exit(1)
	}
}
