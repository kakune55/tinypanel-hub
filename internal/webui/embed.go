package webui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed dist/*
var dist embed.FS

func Handler() http.Handler {
	fsys, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return spaHandler{
		files: http.FileServer(http.FS(fsys)),
		fsys:  fsys,
	}
}

type spaHandler struct {
	files http.Handler
	fsys  fs.FS
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "." || name == "" {
		name = "index.html"
	}
	if _, err := fs.Stat(h.fsys, name); err != nil {
		r = r.Clone(r.Context())
		r.URL.Path = "/"
	}
	h.files.ServeHTTP(w, r)
}
