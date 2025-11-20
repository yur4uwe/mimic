package memserver

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
)

// simple in-memory WebDAV-ish backend used by tests.
// supports PUT (store), GET (full) and Range GET (partial).
type MemBackend struct {
	mu sync.Mutex
	M  map[string][]byte
}

func NewMemBackend() *MemBackend {
	return &MemBackend{M: make(map[string][]byte)}
}

func (b *MemBackend) Reset() {
	b.mu.Lock()
	b.M = make(map[string][]byte)
	b.mu.Unlock()
}

func (b *MemBackend) Set(key string, val []byte) {
	b.mu.Lock()
	b.M[key] = val
	b.mu.Unlock()
}
func (b *MemBackend) Get(key string) ([]byte, bool) {
	b.mu.Lock()
	val, ok := b.M[key]
	b.mu.Unlock()
	return val, ok
}

func (b *MemBackend) handler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	switch r.Method {
	case http.MethodPut:
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "read body", http.StatusInternalServerError)
			return
		}
		b.mu.Lock()
		b.M[path] = body
		b.mu.Unlock()
		// respond like a WebDAV PUT might
		w.WriteHeader(http.StatusCreated)
	case http.MethodGet, http.MethodHead:
		b.mu.Lock()
		data, ok := b.M[path]
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

func NewTestServer() (*httptest.Server, *MemBackend) {
	b := NewMemBackend()
	s := httptest.NewServer(http.HandlerFunc(b.handler))
	return s, b
}
