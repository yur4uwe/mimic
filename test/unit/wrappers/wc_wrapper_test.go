package wrappers

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/mimic/internal/core/cache"
	"github.com/mimic/internal/core/wrappers"
	"github.com/mimic/test/utils/memserver"
)

func newWrapperWithServer(t *testing.T) (*wrappers.WebdavClient, *memserver.MemBackend, func()) {
	t.Helper()
	srv, backend := memserver.NewTestServer()
	cache := cache.NewNodeCache(1*time.Minute, 100)
	wc := wrappers.NewWebdavClient(cache, srv.URL, "", "")
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

	// fetch whole content using public API
	got, err := wc.Read(name)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("unexpected content: got=%q want=%q", got, payload)
	}

	// also verify backend stored it
	stored, ok := backend.Get(name)
	if !ok {
		t.Fatalf("expected backend to have key %s", name)
	}
	if !bytes.Equal(stored, payload) {
		t.Fatalf("backend mismatch: %v", stored)
	}
}

func TestReadRange(t *testing.T) {
	wc, backend, cleanup := newWrapperWithServer(t)
	defer cleanup()

	name := "alphabet.txt"
	data := []byte("abcdefghijklmnopqrstuvwxyz")
	backend.Set(name, data)

	// request first 10 bytes via public ReadRange API
	rc, err := wc.ReadRange(name, 0, 10)
	if err != nil {
		t.Fatalf("ReadRange failed: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading range failed: %v", err)
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
	backend.Set(name, []byte("HelloWorld"))

	// overwrite at offset 5 with "123" -> "Hello123ld"
	if err := wc.WriteOffset(name, []byte("123"), 5); err != nil {
		t.Fatalf("WriteOffset failed: %v", err)
	}
	got, err := wc.Read(name)
	if err != nil {
		t.Fatalf("Read after writeoffset failed: %v", err)
	}
	if !bytes.Equal(got, []byte("Hello123ld")) {
		t.Fatalf("unexpected merged content: %q", got)
	}

	// extend: write at offset beyond current end -> should pad with zeros up to offset then data
	// current length is 10; write "X" at offset 15 -> result length should be 16 with zeros between
	if err := wc.WriteOffset(name, []byte("X"), 15); err != nil {
		t.Fatalf("WriteOffset extend failed: %v", err)
	}
	got2, err := wc.Read(name)
	if err != nil {
		t.Fatalf("Read after extend failed: %v", err)
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
