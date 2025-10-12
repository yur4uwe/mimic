package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/mimic/internal/fs"
)

func main() {
	flag.Parse()

	filesystem := fs.New()
	mountpoint := flag.Arg(0)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, os.Kill)

	if err := filesystem.Mount(mountpoint); err != nil {
		fmt.Println("Mount failed:", err)
		os.Exit(1)
	}

	go func() {
		fmt.Println("Arrived at block")
		<-sig
		if err := filesystem.Unmount(); err != nil {
			fmt.Println("Unmount failed:", err)
		}
		os.Exit(0)
	}()
}
