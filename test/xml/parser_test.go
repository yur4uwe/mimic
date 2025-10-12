package xml_test

import (
	"encoding/xml"
	"testing"

	"github.com/mimic/internal/core/webdav"
)

const sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<D:multistatus xmlns:D="DAV:">
<D:response xmlns:lp1="DAV:" xmlns:lp2="http://apache.org/dav/props/">
<D:href>/</D:href>
<D:propstat>
<D:prop>
<lp1:resourcetype><D:collection/></lp1:resourcetype>
<lp1:creationdate>2025-10-12T12:29:35Z</lp1:creationdate>
<lp1:getlastmodified>Sun, 12 Oct 2025 12:29:35 GMT</lp1:getlastmodified>
<lp1:getetag>"1000-640f54dc19069"</lp1:getetag>
<D:supportedlock>
<D:lockentry>
<D:lockscope><D:exclusive/></D:lockscope>
<D:locktype><D:write/></D:locktype>
</D:lockentry>
<D:lockentry>
<D:lockscope><D:shared/></D:lockscope>
<D:locktype><D:write/></D:locktype>
</D:lockentry>
</D:supportedlock>
<D:lockdiscovery/>
<D:getcontenttype>httpd/unix-directory</D:getcontenttype>
</D:prop>
<D:status>HTTP/1.1 200 OK</D:status>
</D:propstat>
</D:response>
</D:multistatus>`

func TestParse(t *testing.T) {
	expected := webdav.Multistatus{
		Responses: []webdav.Response{
			{
				Href: "/",
				Propstat: []webdav.Propstat{
					{
						Prop: webdav.Prop{
							ResourceType: webdav.ResourceType{
								Collection: &struct{}{},
							},
							CreationDate: "2025-10-12T12:29:35Z",
							LastModified: "Sun, 12 Oct 2025 12:29:35 GMT",
							Etag:         `"1000-640f54dc19069"`,
							ContentType:  "httpd/unix-directory",
						},
						Status: "HTTP/1.1 200 OK",
					},
				},
			},
		},
	}

	var result webdav.Multistatus
	if err := xml.Unmarshal([]byte(sampleXML), &result); err != nil {
		t.Fatalf("Failed to parse XML: %v", err)
	}

	if len(result.Responses) != len(expected.Responses) {
		t.Fatalf("Expected %d responses, got %d", len(expected.Responses), len(result.Responses))
	}

	for i, resp := range result.Responses {
		if resp.Href != expected.Responses[i].Href {
			t.Errorf("Expected href %s, got %s", expected.Responses[i].Href, resp.Href)
		}
		if len(resp.Propstat) != len(expected.Responses[i].Propstat) {
			t.Errorf("Expected %d propstat, got %d", len(expected.Responses[i].Propstat), len(resp.Propstat))
		}
		for j, propstat := range resp.Propstat {
			if propstat.Status != expected.Responses[i].Propstat[j].Status {
				t.Errorf("Expected status %s, got %s", expected.Responses[i].Propstat[j].Status, propstat.Status)
			}
			if propstat.Prop.CreationDate != expected.Responses[i].Propstat[j].Prop.CreationDate {
				t.Errorf("Expected creation date %s, got %s", expected.Responses[i].Propstat[j].Prop.CreationDate, propstat.Prop.CreationDate)
			}
			if propstat.Prop.LastModified != expected.Responses[i].Propstat[j].Prop.LastModified {
				t.Errorf("Expected last modified %s, got %s", expected.Responses[i].Propstat[j].Prop.LastModified, propstat.Prop.LastModified)
			}
			if propstat.Prop.Etag != expected.Responses[i].Propstat[j].Prop.Etag {
				t.Errorf("Expected etag %s, got %s", expected.Responses[i].Propstat[j].Prop.Etag, propstat.Prop.Etag)
			}
			if propstat.Prop.ContentType != expected.Responses[i].Propstat[j].Prop.ContentType {
				t.Errorf("Expected content type %s, got %s", expected.Responses[i].Propstat[j].Prop.ContentType, propstat.Prop.ContentType)
			}
		}
	}
}
