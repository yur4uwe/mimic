package fs

type FS interface {
	Mount(mount_dest string) error
	Unmount() error
}
