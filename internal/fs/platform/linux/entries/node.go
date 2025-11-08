//go:build linux

package entries

import (
	"context"
	"fmt"
	"hash/crc32"
	"log"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/checks"
	"github.com/mimic/internal/interfaces"
)

// Node represents a file or directory backed by WebDAV
type Node struct {
	wc   interfaces.WebClient
	path string
}

func NewNode(wc interfaces.WebClient, path string) *Node {
	return &Node{
		wc:   wc,
		path: path,
	}
}

func (n *Node) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Println("Attr called for path:", n.path)

	if n.path == "/" {
		a.Mode = os.ModeDir | 0o755
		a.Size = 0
		a.Valid = time.Minute

		return nil
	}

	fi, err := n.wc.Stat(n.path)
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		return err
	}

	attr := casters.FileInfoCast(fi)

	attr.Inode = uint64(crc32.ChecksumIEEE([]byte(n.path)) + 1)

	*a = *attr
	return nil
}

func (n *Node) Lookup(ctx context.Context, name string) (fs.Node, error) {
	childPath := path.Join(n.path, name)
	fmt.Println("Lookup called for path:", childPath)

	fi, err := n.wc.Stat(childPath)
	if err != nil {
		return nil, syscall.Errno(syscall.ENOENT)
	}

	if checks.IsNilInterface(fi) {
		return nil, syscall.Errno(syscall.ENOENT)
	}

	if fi.IsDir() {
		return &Node{wc: n.wc, path: childPath}, nil
	}

	return &File{path: childPath, wc: n.wc}, nil
}

func (n *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll called for path:", n.path)

	infos, err := n.wc.ReadDir(n.path)
	if err != nil {
		return nil, err
	}

	var entries []fuse.Dirent
	for _, fi := range infos {
		if fi == nil {
			continue
		}
		name := fi.Name()

		if strings.ContainsAny(name, " \t\n\r") {
			name = path.Base(name)
		}

		childPath := path.Join(n.path, name)

		var t fuse.DirentType
		if fi.IsDir() {
			t = fuse.DT_Dir
		} else {
			t = fuse.DT_File
		}

		entries = append(entries, fuse.Dirent{
			Inode: uint64(crc32.ChecksumIEEE([]byte(childPath)) + 1),
			Name:  fi.Name(),
			Type:  t,
		})
	}

	return entries, nil
}
