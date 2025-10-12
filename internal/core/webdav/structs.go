package webdav

import "encoding/xml"

type Multistatus struct {
	XMLName   xml.Name   `xml:"multistatus"` // Match the local name only
	Responses []Response `xml:"response"`
}

type Response struct {
	Href     string     `xml:"href"`
	Propstat []Propstat `xml:"propstat"`
}

type Propstat struct {
	Prop   Prop   `xml:"prop"`
	Status string `xml:"status"`
}

type Prop struct {
	ResourceType  ResourceType `xml:"resourcetype"`
	CreationDate  string       `xml:"creationdate"`
	LastModified  string       `xml:"getlastmodified"`
	Etag          string       `xml:"getetag"`
	ContentType   string       `xml:"getcontenttype"`
	ContentLength int64        `xml:"getcontentlength"`
}

type ResourceType struct {
	Collection *struct{} `xml:"collection"`
}
