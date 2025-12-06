package flags

import (
	"strings"

	"github.com/winfsp/cgofuse/fuse"
)

type OpenFlag uint32

func (f OpenFlag) WriteAllowed() bool {
	// write allowed if O_WRONLY or O_RDWR is set
	return (uint32(f)&uint32(fuse.O_WRONLY) != 0) || (uint32(f)&uint32(fuse.O_RDWR) != 0)
}

func (f OpenFlag) ReadAllowed() bool {
	// read allowed unless open is explicitly write-only
	return uint32(f)&uint32(fuse.O_WRONLY) == 0
}

func (f OpenFlag) Append() bool {
	return uint32(f)&uint32(fuse.O_APPEND) != 0
}

func (f OpenFlag) Create() bool {
	return uint32(f)&uint32(fuse.O_CREAT) != 0
}

func (f OpenFlag) Truncate() bool {
	return uint32(f)&uint32(fuse.O_TRUNC) != 0
}

func (f OpenFlag) Exclusive() bool {
	return uint32(f)&uint32(fuse.O_EXCL) != 0
}

func (f OpenFlag) String() string {
	flags := []string{}
	if f.ReadAllowed() && f.WriteAllowed() {
		flags = append(flags, "O_RDWR")
	} else if f.ReadAllowed() {
		flags = append(flags, "O_RDONLY")
	} else if f.WriteAllowed() {
		flags = append(flags, "O_WRONLY")
	}
	if f.Append() {
		flags = append(flags, "O_APPEND")
	}
	if f.Create() {
		flags = append(flags, "O_CREATE")
	}
	if f.Truncate() {
		flags = append(flags, "O_TRUNC")
	}
	if f.Exclusive() {
		flags = append(flags, "O_EXCL")
	}
	return strings.Join(flags, "|")
}
