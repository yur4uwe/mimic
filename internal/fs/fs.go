package fs

type FS interface {
	Mount(mount_dest string, mflags []string) error
	Unmount() error
}
