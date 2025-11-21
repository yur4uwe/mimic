//go:build linux

package entries

import (
	"context"
	"errors"
	"hash/crc32"
	"os"
	"path"
	"strings"
	"syscall"
	"time"

	"bazil.org/fuse"
	"bazil.org/fuse/fs"
	"github.com/mimic/internal/core/casters"
	"github.com/mimic/internal/core/checks"
	"github.com/mimic/internal/core/flags"
	"github.com/mimic/internal/core/logger"
	"github.com/mimic/internal/interfaces"
)

type Node struct {
	wc     interfaces.WebClient
	logger logger.FullLogger
	path   string
}

func NewNode(wc interfaces.WebClient, logger logger.FullLogger, path string) *Node {
	return &Node{
		wc:     wc,
		logger: logger,
		path:   path,
	}
}

func (n *Node) Attr(ctx context.Context, a *fuse.Attr) error {
	n.logger.Logf("[Attr] called for path: %s", n.path)

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

func (n *Node) Access(ctx context.Context, req *fuse.AccessRequest) error {
	n.logger.Logf("[Access] called for path: %s with mode: %d", n.path, req.Mask)
	return nil
}

func (n *Node) Lookup(ctx context.Context, name string) (fs.Node, error) {
	childPath := path.Join(n.path, name)
	n.logger.Logf("[Lookup] called for path: %s", childPath)

	fi, err := n.wc.Stat(childPath)
	if err != nil {
		return nil, syscall.Errno(syscall.ENOENT)
	}

	if checks.IsNilInterface(fi) {
		return nil, syscall.Errno(syscall.ENOENT)
	}

	return NewNode(n.wc, n.logger, childPath), nil
}

func (n *Node) Poll(ctx context.Context, req *fuse.PollRequest, resp *fuse.PollResponse) error {
	return syscall.Errno(syscall.ENOSYS)
}

func (n *Node) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	n.logger.Logf("[ReadDirAll] called for path: %s", n.path)

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
	n.logger.Logf("[Mkdir] called for path: %s", newDirPath)
	if err := n.wc.Mkdir(newDirPath, req.Mode); err != nil {
		return nil, err
	}

	return NewNode(n.wc, n.logger, newDirPath), nil
}

func (n *Node) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
	newFilePath := path.Join(n.path, req.Name)
	oflags := flags.OpenFlag(uint32(req.Flags))
	n.logger.Logf("[Create] called for path: %s, with flags: %v", newFilePath, oflags)

	if err := n.wc.Create(newFilePath); err != nil {
		return nil, nil, err
	}

	handle := NewHandle(n.wc, n.logger, newFilePath, oflags)
	node := NewNode(n.wc, n.logger, newFilePath)
	return node, handle, nil
}

func (n *Node) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	targetPath := path.Join(n.path, req.Name)
	n.logger.Logf("[Remove] called for path: %s", targetPath)

	if req.Dir {
		if err := n.wc.Rmdir(targetPath); err != nil {
			return err
		}
	} else {
		if err := n.wc.Remove(targetPath); err != nil {
			return err
		}
	}

	return nil
}

func (n *Node) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	oldPath := path.Join(n.path, req.OldName)
	newNode, ok := newDir.(*Node)
	if !ok {
		return errors.New("invalid target directory")
	}

	newPath := path.Join(newNode.path, req.NewName)
	n.logger.Logf("Rename called from path: %s to path: %s", oldPath, newPath)
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
	flags := flags.OpenFlag(uint32(req.Flags))
	n.logger.Logf("[Open] called for path: %s, with flags: %v", n.path, flags)

	if _, err := n.wc.Stat(n.path); flags.Create() && os.IsNotExist(err) {
		if err := n.wc.Create(n.path); err != nil {
			return nil, err
		}
	}

	handle := NewHandle(n.wc, n.logger, n.path, flags)

	return handle, nil
}
