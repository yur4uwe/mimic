package fs

// Error codes
// Headers are incorrectly imported on Windows
// So i defined them here
const (
	EPERM   = 1
	ENOENT  = 2
	EIO     = 5
	EACCES  = 13
	EEXIST  = 17
	ENOTDIR = 20
	EINVAL  = 22
	ENOSYS  = 38
)
