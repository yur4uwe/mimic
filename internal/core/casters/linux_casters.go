//go:build linux

package casters

import (
	"os"

	"bazil.org/fuse"
)

func FileInfoCast(f os.FileInfo) *fuse.Attr {
	return nil
}
