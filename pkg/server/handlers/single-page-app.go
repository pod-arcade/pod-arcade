package handlers

import (
	"io"
	"io/fs"
	"net/http"
)

type SinglePageAppHandler struct {
	// This path is relative to the static path
	IndexPath  string
	FS         http.FileSystem
	FileServer http.Handler
}

func NewSinglePageAppHandler(fileSystem fs.FS, indexPath string, staticPath string) (*SinglePageAppHandler, error) {
	subFs, err := fs.Sub(fileSystem, staticPath)
	if err != nil {
		return nil, err
	}
	httpFs := http.FS(subFs)
	return &SinglePageAppHandler{
		IndexPath:  indexPath,
		FS:         httpFs,
		FileServer: http.FileServer(httpFs),
	}, nil
}

func (h *SinglePageAppHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	f, err := h.FS.Open(path)
	if err != nil {
		h.serveIndex(w, r)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		h.serveIndex(w, r)
		return
	}

	if fi.IsDir() {
		h.serveIndex(w, r)
		return
	}

	h.FileServer.ServeHTTP(w, r)
}

// serveIndex serves the index file from FS.
func (h *SinglePageAppHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	indexFile, err := h.FS.Open(h.IndexPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer indexFile.Close()

	// Copy the index file's content to the response body
	_, err = io.Copy(w, indexFile)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
