//go:build windows

package fs

import (
	"fmt"
	"os"
	"strings"

	"github.com/mimic/internal/core/webdav"
	"github.com/winfsp/cgofuse/fuse"
)

type winfspFS struct {
	fuse.FileSystemBase
	wc *webdav.Client
}

func New(webdavClient *webdav.Client) FS {
	return &winfspFS{
		wc: webdavClient,
	}
}

func (f *winfspFS) Mount(mountpoint string) error {
	fmt.Println("Mounting WinFSP filesystem at", os.Args[1:])

	host := fuse.NewFileSystemHost(f)
	if !host.Mount("", os.Args[1:]) {
		return fmt.Errorf("failed to mount WinFSP filesystem")
	}

	return nil
}

func (f *winfspFS) Unmount() error {
	fmt.Println("Unmounting WinFSP filesystem")
	return nil
}

func (f *winfspFS) Getattr(path string, stat *fuse.Stat_t, fh uint64) int {
	if path != "/" {
		path = strings.TrimPrefix(path, "/")
	}
	fmt.Println("Getattr called for path:", path)

	props, err := f.wc.Props(path)
	if err != nil {
		fmt.Println("Error getting properties:", err)
		return fuse.ENOENT
	}

	if props.ResourceType.Collection != nil {
		stat.Mode = fuse.S_IFDIR | 0755
	} else {
		stat.Mode = fuse.S_IFREG | 0644
		stat.Size = props.ContentLength
	}

	return 0
}

func (f *winfspFS) Readdir(path string, fill func(string, *fuse.Stat_t, int64) bool, off int64, fh uint64) int {
	fmt.Println("Readdir called for path:", path)

	items, err := f.wc.List(path)
	if err != nil {
		fmt.Println("Error listing directory:", err)
		return fuse.ENOENT
	}

	for _, item := range items {
		fill(item, nil, 0)
	}

	return 0
}

func (f *winfspFS) Open(path string, flags int) (int, uint64) {
	fmt.Println("Open called for path:", path)
	return 0, 0
}

func (f *winfspFS) Read(path string, buff []byte, ofst int64, fh uint64) int {
	fmt.Println("Read called for path:", path)
	data := "Hello, World!"
	copy(buff, data)
	return len(data)
}

func (f *winfspFS) Write(path string, buff []byte, ofst int64, fh uint64) int {
	fmt.Println("Write called for path:", path)
	fmt.Printf("Data written: %s\n", string(buff))
	return len(buff)
}

func (f *winfspFS) Create(string, int, uint32) (int, uint64) {
	fmt.Println("Create called")
	return 0, 0
}
func (f *winfspFS) Unlink(path string) int {
	fmt.Println("Unlink called for path:", path)
	return 0
}

func (f *winfspFS) Mkdir(path string, mode uint32) int {
	fmt.Println("Mkdir called for path:", path)
	return 0
}

func (f *winfspFS) Rmdir(path string) int {
	fmt.Println("Rmdir called for path:", path)
	return 0
}

func (f *winfspFS) Rename(oldPath string, newPath string) int {
	fmt.Println("Rename called from", oldPath, "to", newPath)
	return 0
}
