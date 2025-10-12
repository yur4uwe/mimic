package webdav

import (
	"encoding/xml"
	"fmt"
	"net/http"
)

// This is package for interacting with a ready WebDAV server.

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

func (c *Client) HealthCheck() error {
	return nil
}

func (c *Client) List(path string) ([]string, error) {
	fmt.Println("Listing directory at", c.Server+path)

	req, err := http.NewRequest(PROPFIND, c.Server+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.User, c.Pass)
	req.Header.Set("Depth", "1")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != StatusMultiStatus {
		return nil, fmt.Errorf("failed to list files: %s", resp.Status)
	}

	var multistatus Multistatus
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&multistatus); err != nil {
		return nil, err
	}

	var items []string
	for _, response := range multistatus.Responses {
		items = append(items, response.Href)
	}

	fmt.Println("Listed items:", items)

	return items, nil
}

func (c *Client) Post(path string, data []byte) error {
	return nil
}

func (c *Client) Get(path string) ([]byte, error) {
	return nil, nil
}

func (c *Client) Delete(path string) error {
	return nil
}

func (c *Client) Put(data []byte, path string) error {
	return nil
}

func (c *Client) Move(srcPath, destPath string) error {
	return nil
}

func (c *Client) Props(path string) (*Prop, error) {
	fmt.Println("Propfind called for", c.Server+path)

	req, err := http.NewRequest(PROPFIND, c.Server+path, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.User, c.Pass)
	req.Header.Set("Depth", "0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != StatusMultiStatus {
		return nil, fmt.Errorf("failed to propfind: %s", resp.Status)
	}

	var res Multistatus
	decoder := xml.NewDecoder(resp.Body)
	if err := decoder.Decode(&res); err != nil {
		return nil, err
	}

	fmt.Println("Propfind successful for", path, "with properties:", res)

	return &res.Responses[0].Propstat[0].Prop, nil
}
