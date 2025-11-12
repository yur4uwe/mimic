package interfaces

import (
	"io"
	"os"
)

// WebClient defines the minimal operations the FS needs from a
// WebDAV-like client. Implementations (gowebdav wrapper or mocks)
// should provide all methods below.
type WebClient interface {
	// Metadata and directory listing
	Stat(name string) (os.FileInfo, error)
	ReadDir(name string) ([]os.FileInfo, error)

	// Read helpers
	Read(name string) ([]byte, error)              // read whole file
	ReadStream(name string) (io.ReadCloser, error) // streaming read
	ReadRange(name string, offset, length int64) (io.ReadCloser, error)

	// Write
	Write(name string, data []byte) error // write/overwrite with byte slice
	WriteOffset(name string, data []byte, offset int64) error

	// create / remove
	Create(name string) error                  // create new file with data (can alias Write)
	Remove(name string) error                  // delete a file
	Truncate(name string, size int64) error    // truncate or extend a file to given size
	Mkdir(name string, mode os.FileMode) error // create directory
	Rmdir(name string) error                   // remove directory
	Rename(oldname, newname string) error      // rename/move
}
