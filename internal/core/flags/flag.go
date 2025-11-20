package flags

import "os"

type OpenFlag uint32

func (f OpenFlag) WriteAllowed() bool {
	// write allowed unless open is explicitly read-only
	return uint32(f)&uint32(os.O_RDONLY) == 0
}

func (f OpenFlag) ReadAllowed() bool {
	// read allowed unless open is explicitly write-only
	return uint32(f)&uint32(os.O_WRONLY) == 0
}

func (f OpenFlag) Append() bool {
	return uint32(f)&uint32(os.O_APPEND) != 0
}

func (f OpenFlag) Create() bool {
	return uint32(f)&uint32(os.O_CREATE) != 0
}

func (f OpenFlag) Truncate() bool {
	return uint32(f)&uint32(os.O_TRUNC) != 0
}

func (f OpenFlag) Exclusive() bool {
	return uint32(f)&uint32(os.O_EXCL) != 0
}
