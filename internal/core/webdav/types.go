package webdav

import (
	"time"

	restypes "github.com/mimic/internal/core/webdav/response_types"
)

type Client struct {
	Server string
	User   string
	Pass   string
}

func NewClient(server, user, pass string) *Client {
	return &Client{
		Server: server,
		User:   user,
		Pass:   pass,
	}
}

type FileInfo struct {
	Name         string    // Name of the file or directory
	IsDir        bool      // Whether it's a directory
	Size         int64     // Size of the file (0 for directories)
	CreationDate time.Time // Creation date
	LastModified time.Time // Last modified date
	Etag         string    // ETag
	ContentType  string    // Content type (e.g., "text/plain")
}

func ToFileInfo(entry restypes.Response) FileInfo {
	cd, _ := time.Parse(time.RFC3339, entry.Propstat[0].Prop.CreationDate)
	md, _ := time.Parse(time.RFC1123, entry.Propstat[0].Prop.LastModified)

	return FileInfo{
		Name:         entry.Href,
		IsDir:        entry.Propstat[0].Prop.ResourceType.Collection != nil,
		Size:         entry.Propstat[0].Prop.ContentLength,
		CreationDate: cd,
		LastModified: md,
		Etag:         entry.Propstat[0].Prop.Etag,
		ContentType:  entry.Propstat[0].Prop.ContentType,
	}
}
