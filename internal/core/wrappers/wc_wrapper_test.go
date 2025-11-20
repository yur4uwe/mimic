package wrappers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mimic/internal/core/cache"
	"github.com/studio-b12/gowebdav"
)

// simple in-memory WebDAV-ish backend used by tests.
// supports PUT (store), GET (full) and Range GET (partial).
type memBackend struct {
	mu sync.Mutex
	m  map[string][]byte
}

func newMemBackend() *memBackend {
	return &memBackend{m: make(map[string][]byte)}
}

func (b *memBackend) handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusInternalServerError)
			return
		}
		b.mu.Lock()
		b.m[path] = body
		b.mu.Unlock()
		// respond like a WebDAV PUT might
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet, http.MethodHead:
		b.mu.Lock()
		data, ok := b.m[path]
		b.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}

		// Range support
		if rng := r.Header.Get("Range"); rng != "" {
			// expect "bytes=start-end" or "bytes=start-"
			if !strings.HasPrefix(rng, "bytes=") {
				http.Error(w, "invalid range", http.StatusRequestedRangeNotSatisfiable)
				return
			}
			rng = strings.TrimPrefix(rng, "bytes=")
			parts := strings.SplitN(rng, "-", 2)
			start, _ := strconv.ParseInt(parts[0], 10, 64)
			var end int64 = int64(len(data)) - 1
			if parts[1] != "" {
				e, err := strconv.ParseInt(parts[1], 10, 64)
				if err == nil {
					end = e
				}
			}
			if start < 0 || start > end || start >= int64(len(data)) {
				http.Error(w, "range not satisfiable", http.StatusRequestedRangeNotSatisfiable)
				return
			}
			if end >= int64(len(data)) {
				end = int64(len(data)) - 1
			}
			part := data[start : end+1]
			w.Header().Set("Content-Range", "bytes "+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10)+"/"+strconv.FormatInt(int64(len(data)), 10))
			w.Header().Set("Content-Length", strconv.Itoa(len(part)))
			w.WriteHeader(http.StatusPartialContent)
			if r.Method == http.MethodGet {
				_, _ = w.Write(part)
			}
			return
		}

		// full GET
		w.Header().Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	default:
		http.Error(w, "not implemented", http.StatusNotImplemented)
	}
}

func newTestServer() (*httptest.Server, *memBackend) {
	b := newMemBackend()
	s := httptest.NewServer(http.HandlerFunc(b.handler))
	return s, b
}

func newWrapperWithServer(t *testing.T) (*WebdavClient, *memBackend, func()) {
	t.Helper()
	srv, backend := newTestServer()
	c := gowebdav.NewClient(srv.URL, "", "")
	// allow client a bit for any internal setup
	c.SetTimeout(5 * time.Second)
	cache := cache.NewNodeCache(1*time.Minute, 100)
	wc := NewWebdavClient(c, cache)
	cleanup := func() { srv.Close() }
	return wc, backend, cleanup
}

func TestWriteAndFetchSmall(t *testing.T) {
	wc, backend, cleanup := newWrapperWithServer(t)
	defer cleanup()

	name := "small.txt"
	payload := []byte("hello-world")

	if err := wc.Write(name, payload); err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// fetch whole content
	got, err := wc.fetch(name, 0)
	if err != nil {
		t.Fatalf("fetch failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected content: got=%q want=%q", got, payload)
	}

	// also verify backend stored it
	backend.mu.Lock()
	stored := backend.m[name]
	backend.mu.Unlock()
	if !bytes.Equal(stored, payload) {
		t.Fatalf("backend mismatch: %v", stored)
	}
}

func TestFetchRange(t *testing.T) {
	wc, backend, cleanup := newWrapperWithServer(t)
	defer cleanup()

	name := "alphabet.txt"
	data := []byte("abcdefghijklmnopqrstuvwxyz")
	backend.mu.Lock()
	backend.m[name] = data
	backend.mu.Unlock()

	// request first 10 bytes
	got, err := wc.fetch(name, 10)
	if err != nil {
		t.Fatalf("fetch range failed: %v", err)
	}
	if len(got) != 10 {
		t.Fatalf("expected 10 bytes, got %d", len(got))
	}
	if !bytes.Equal(got, data[:10]) {
		t.Fatalf("range mismatch: got=%q want=%q", got, data[:10])
	}
}

func TestWriteOffsetMergeAndExtend(t *testing.T) {
	wc, backend, cleanup := newWrapperWithServer(t)
	defer cleanup()

	name := "merge.txt"
	// initial content: "HelloWorld"
	backend.mu.Lock()
	backend.m[name] = []byte("HelloWorld")
	backend.mu.Unlock()

	// overwrite at offset 5 with "123" -> "Hello123ld"
	if err := wc.WriteOffset(name, []byte("123"), 5); err != nil {
		t.Fatalf("WriteOffset failed: %v", err)
	}
	got, err := wc.fetch(name, 0)
	if err != nil {
		t.Fatalf("fetch after writeoffset failed: %v", err)
	}
	if !bytes.Equal(got, []byte("Hello123ld")) {
		t.Fatalf("unexpected merged content: %q", got)
	}

	// extend: write at offset beyond current end -> should pad with zeros up to offset then data
	// current length is 10; write "X" at offset 15 -> result length should be 16 with zeros between
	if err := wc.WriteOffset(name, []byte("X"), 15); err != nil {
		t.Fatalf("WriteOffset extend failed: %v", err)
	}
	got2, err := wc.fetch(name, 0)
	if err != nil {
		t.Fatalf("fetch after extend failed: %v", err)
	}
	if len(got2) != 16 {
		t.Fatalf("unexpected length after extend: %d", len(got2))
	}
	if got2[15] != 'X' {
		t.Fatalf("expected placed byte at offset 15, got %v", got2[15])
	}
}

func TestCreateRejectsDirPath(t *testing.T) {
	wc, _, cleanup := newWrapperWithServer(t)
	defer cleanup()

	if err := wc.Create("some/dir/"); err == nil {
		t.Fatalf("expected error when creating with trailing slash")
	}
}
