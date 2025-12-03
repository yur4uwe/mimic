package wrappers

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func buildURL(baseURL, name string) string {
	base := strings.TrimRight(baseURL, "/")
	path := strings.TrimLeft(name, "/")
	return base + "/" + path
}

func davRequest(method, url, uname, pass string, body io.Reader, headers map[string]string) (int, []byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	req.SetBasicAuth(uname, pass)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	return resp.StatusCode, data, nil
}

func (w *WebdavClient) commit(name string, data []byte) error {
	defer w.cache.Invalidate(name)
	if len(data) > streamThreshold {
		return w.client.WriteStream(name, bytes.NewReader(data), 0644)
	} else {
		return w.client.Write(name, data, 0644)
	}
}

// tryPartialPut attempts a non-standard partial PUT using Content-Range header.
// Returns (true, nil) if the server accepted the partial update (2xx),
// (false, nil) if server rejected (non-2xx), or (false, err) on network error.
func (w *WebdavClient) tryPartialPut(name string, offset int64, data []byte) (bool, error) {
	url := buildURL(w.baseURL, name)

	// Content-Range: bytes <start>-<end>
	end := offset + int64(len(data)) - 1
	crange := fmt.Sprintf("bytes %d-%d/*", offset, end)
	headers := map[string]string{
		"Content-Range": crange,
	}

	code, _, err := davRequest("PUT", url, w.username, w.password, bytes.NewReader(data), headers)
	if err != nil {
		return false, err
	}

	if code >= 200 && code < 300 {
		return true, nil
	}
	return false, nil
}

func (w *WebdavClient) fetch(name string) ([]byte, error) {
	if strings.HasSuffix(name, "/") && name != "/" {
		name = strings.TrimSuffix(name, "/")
	}

	if rc, err := w.client.ReadStream(name); err == nil {
		defer rc.Close()
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		return data, nil
	}

	all, err := w.client.Read(name)
	if err != nil {
		return nil, err
	}
	return all, nil
}
