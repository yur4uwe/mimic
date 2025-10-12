package webdav

import (
	"encoding/xml"
	"fmt"
	"net/http"

	restypes "github.com/mimic/internal/core/webdav/response_types"
)

func (c *Client) ReadDir(path string) ([]FileInfo, error) {
	req, err := http.NewRequest(PROPFIND, c.Server+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.User, c.Pass)
	req.Header.Set(DepthHeader, "1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != StatusMultiStatus {
		return nil, fmt.Errorf("failed to list files: %s", resp.Status)
	}

	var multistatus restypes.Multistatus
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&multistatus); err != nil {
		return nil, err
	}

	var items []FileInfo
	for _, response := range multistatus.Responses {
		items = append(items, ToFileInfo(response))
	}

	return items, nil
}

func (c *Client) GetProps(path string) (*FileInfo, error) {
	req, err := http.NewRequest(PROPFIND, c.Server+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.User, c.Pass)
	req.Header.Set(DepthHeader, "0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != StatusMultiStatus {
		return nil, fmt.Errorf("failed to propfind: %s", resp.Status)
	}

	var xmlresp restypes.Multistatus
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&xmlresp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %v", err)
	}

	res := ToFileInfo(xmlresp.Responses[0])
	return &res, nil
}
