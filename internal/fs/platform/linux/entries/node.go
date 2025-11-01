//go:build linux || darwin

package entries

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/core/casters"
	"github.com/studio-b12/gowebdav"
)

// Node represents a file or directory backed by WebDAV
type Node struct {
	wc   *gowebdav.Client
	path string // absolute path as seen by WebDAV client (leading '/')
}

func NewNode(wc *gowebdav.Client, path string) *Node {
	return &Node{
		wc:   wc,
		path: path,
	}
}

// Attr implements fs.Node
func (n *Node) Attr(ctx context.Context, a *fuse.Attr) error {
	fmt.Println("Attr called for path:", n.path)

	fi, err := n.wc.Stat(n.path)
	if err != nil {
		if os.IsNotExist(err) {
			return syscall.Errno(syscall.ENOENT)
		}
		return err
	}
	attr := casters.FileInfoCast(fi)
	*a = *attr
	return nil
}

// Lookup implements fs.NodeStringLookuper
func (n *Node) Lookup(ctx context.Context, name string) (fs.Node, error) {
	childPath := path.Join(n.path, name)

	fmt.Println("Lookup called for path:", childPath)

retry:
	_, err := n.wc.Stat(childPath)
	if err != nil {
		if !strings.HasSuffix(childPath, "/") {
			childPath += "/"
			goto retry
		}

		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
	}
	return &Node{wc: n.wc, path: childPath}, nil
}

func (n *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	fmt.Println("ReadDirAll called for path:", n.path)

	infos, err := n.wc.ReadDir(n.path)
	if err != nil {
		return nil, err
	}
	var entries []fuse.Dirent
	for _, fi := range infos {
		var t fuse.DirentType
		if fi.IsDir() {
			t = fuse.DT_Dir
		} else {
			t = fuse.DT_File
		}
		entries = append(entries, fuse.Dirent{
			Name: fi.Name(),
			Type: t,
		})
	}
	return entries, nil
}

// File Node behavior: implement ReadAll for convenience
type fileHandle struct {
	path string
	wc   *gowebdav.Client
}

func (n *Node) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// for directories, bazil will not call Open; this is for files
	return &fileHandle{path: n.path, wc: n.wc}, nil
}

func (fh *fileHandle) ReadAll(ctx context.Context) ([]byte, error) {
	// gowebdav.Client has a Read method that returns []byte
	data, err := fh.wc.Read(fh.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
		return nil, err
	}
	return data, nil
}
