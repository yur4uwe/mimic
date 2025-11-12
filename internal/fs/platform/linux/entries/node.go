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

	if checks.IsNilInterface(fi) {
		return syscall.Errno(syscall.ENOENT)
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

	return &Handle{path: childPath, wc: n.wc}, nil
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

func (n *Node) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	newDirPath := path.Join(n.path, req.Name)
	fmt.Println("Mkdir called for path:", newDirPath)
	if err := n.wc.Mkdir(newDirPath, req.Mode); err != nil {
		return nil, err
	}

	return &Node{wc: n.wc, path: newDirPath}, nil
}

func (n *Node) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	newFilePath := path.Join(n.path, req.Name)
	fmt.Println("Create called for path:", newFilePath)

	if err := n.wc.Create(newFilePath); err != nil {
		return nil, nil, err
	}

	handle := &Handle{path: newFilePath, wc: n.wc}
	node := &Node{wc: n.wc, path: newFilePath}
	return node, handle, nil
}

func (n *Node) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	targetPath := path.Join(n.path, req.Name)
	fmt.Println("Remove called for path:", targetPath)

	if err := n.wc.Remove(targetPath); err != nil {
		return err
	}

	return nil
}

func (n *Node) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	oldPath := path.Join(n.path, req.OldName)
	newNode, ok := newDir.(*Node)
	if !ok {
		return fmt.Errorf("invalid target directory")
	}

	newPath := path.Join(newNode.path, req.NewName)
	fmt.Println("Rename called from path:", oldPath, "to path:", newPath)
	if err := n.wc.Rename(oldPath, newPath); err != nil {
		return err
	}

	return nil
}

func (n *Node) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if !req.Valid.Size() {
		return nil
	}

	if err := n.wc.Truncate(n.path, int64(req.Size)); err != nil {
		return err
	}

	if fi, err := n.wc.Stat(n.path); err == nil {
		attr := casters.FileInfoCast(fi)
		*resp = fuse.SetattrResponse{Attr: *attr}
	}

	return nil
}

func (n *Node) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fmt.Println("Open called for path:", n.path)

	handle := &Handle{path: n.path, wc: n.wc}

	return handle, nil
}
