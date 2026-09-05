// Code generated on 2026-09-05T20:01:05.3NZ:z, DO NOT EDIT.
package dist

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//go:embed *.html *.css *.js *.wasm *.ico audio/*.wav *.json icons/*.png external/*
var embeddedFsys embed.FS

var _ http.File = httpFile{}

var _ fs.FileInfo = httpFileInfo{}

var hashMap = func() map[string]string {
	hashes := make(map[string]string)

	var readDir func(string) error
	readDir = func(name string) error {
		entries, err := embeddedFsys.ReadDir(name)
		if err != nil {
			return err
		}

		for _, entry := range entries {
			path := filepath.Join(name, entry.Name())
			if entry.IsDir() {
				if err := readDir(path); err != nil {
					return err
				}

			} else {
				content, err := embeddedFsys.ReadFile(path)
				if err != nil {
					return err
				}

				hash := sha256.Sum256(content)
				hashes[path] = hex.EncodeToString(hash[:])

			}
		}

		return nil
	}

	if err := readDir("."); err != nil {
		log.Fatal(err)
	}

	return hashes
}()

var HttpFS interface {
	http.FileSystem
	FS() fs.FS
} = &httpFS{fsys: embeddedFsys}

type httpFile struct {
	name   string
	size   int64
	reader io.ReadSeeker
}

func (h httpFile) Close() error                                 { return nil }
func (h httpFile) Readdir(_ int) ([]fs.FileInfo, error)         { return nil, fs.ErrNotExist }
func (h httpFile) Read(p []byte) (n int, err error)             { return h.reader.Read(p) }
func (h httpFile) Seek(offset int64, whence int) (int64, error) { return h.reader.Seek(offset, whence) }
func (h httpFile) Stat() (fs.FileInfo, error)                   { return httpFileInfo{name: h.name, size: h.size}, nil }

type httpFileInfo struct {
	name string
	size int64
}

func (h httpFileInfo) Name() string     { return filepath.Base(h.name) }
func (h httpFileInfo) Size() int64      { return h.size }
func (httpFileInfo) Mode() fs.FileMode  { return fs.FileMode(os.ModePerm) }
func (httpFileInfo) ModTime() time.Time { return time.Time{} }
func (httpFileInfo) IsDir() bool        { return false }
func (httpFileInfo) Sys() any           { return nil }

type httpFS struct{ fsys fs.FS }

func (h *httpFS) FS() fs.FS { return h.fsys }

func (h *httpFS) Open(name string) (http.File, error) {
	if !strings.HasSuffix(strings.TrimSuffix(name, "/"), ".sha256") {
		return http.FS(h.fsys).Open(name)
	}

	if hash, ok := hashMap[strings.TrimSuffix(filepath.Base(name), ".sha256")]; ok {
		return httpFile{
			name:   name,
			size:   int64(len(hash)),
			reader: strings.NewReader(hash),
		}, nil
	}

	return nil, fs.ErrNotExist
}

func BuildTime() string { return "2026-09-05T20:01:05.3NZ:z" }

func LookupHash(name string) (string, bool) {
	hash, ok := hashMap[name]
	return hash, ok
}
