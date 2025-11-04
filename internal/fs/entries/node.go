//go:build linux || darwin

package entries

import (
	"context"
	"os"
	"path"
	"strings"
	"syscall"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/interfaces"
)

// Node represents a file or directory backed by WebDAV
type Node struct {
	client interfaces.WebClient
	path   string // absolute path as seen by WebDAV client (leading '/')
}

func NewNode(client interfaces.WebClient, path string) *Node {
	return &Node{
		client: client,
		path:   path,
	}
}

// Attr implements fs.Node
func (n *Node) Attr(ctx context.Context, a *fuse.Attr) error {
	fi, err := n.client.Stat(n.path)
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
	// ensure path form matches WebDAV expectations (keep trailing slash only when dir)
	// Stat will tell us if it exists
retry:
	_, err := n.client.Stat(childPath)
	if err != nil {
		// try with trailing slash if not found
		if !strings.HasSuffix(childPath, "/") {
			childPath += "/"
			goto retry
		}

		if os.IsNotExist(err) {
			return nil, syscall.Errno(syscall.ENOENT)
		}
	}
	return &Node{client: n.client, path: childPath}, nil
}

// ReadDirAll implements fs.HandleReadDirAller / fs.NodeReaddirer
func (n *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	infos, err := n.client.ReadDir(n.path)
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
	path   string
	client interfaces.WebClient
}

func (n *Node) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	// for directories, bazil will not call Open; this is for files
	return &fileHandle{path: n.path, client: n.client}, nil
}

func (fh *fileHandle) ReadAll(ctx context.Context) ([]byte, error) {
	// gowebdav.Client has a Read method that returns []byte
	data, err := fh.client.Read(fh.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fuse.ENOENT
		}
		return nil, err
	}
	return data, nil
}
