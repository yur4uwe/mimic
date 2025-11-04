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
	Read(name string) ([]byte, error)                                         // read whole file
	ReadStream(name string) (io.ReadCloser, error)                            // streaming read
	ReadStreamRange(name string, offset, length int64) (io.ReadCloser, error) // ranged streaming read

	// Write / create / remove
	Write(name string, data []byte) error  // write/overwrite with byte slice
	Create(name string, data []byte) error // create new file with data (can alias Write)
	Remove(name string) error              // delete a file
	Mkdir(name string) error               // create directory
	Rmdir(name string) error               // remove directory
	Rename(oldname, newname string) error  // rename/move
}
