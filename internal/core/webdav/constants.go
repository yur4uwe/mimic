package webdav

// HTTP methods used in WebDAV protocol
const (
	PROPFIND = "PROPFIND"
	MKCOL    = "MKCOL"
	MOVE     = "MOVE"
	COPY     = "COPY"
	LOCK     = "LOCK"
	UNLOCK   = "UNLOCK"
	OPTIONS  = "OPTIONS"
)

// HTTP status codes used in WebDAV protocol
const (
	StatusMultiStatus         = 207
	StatusUnprocessableEntity = 422
	StatusLocked              = 423
	StatusFailedDependency    = 424
	StatusInsufficientStorage = 507
)

// HTTP headers used in WebDAV protocol
const (
	DepthHeader = "Depth"
)
