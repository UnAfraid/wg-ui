package frontend

import (
	"embed"
	"io/fs"
	"net/http"
	"os"
)

//go:embed app/*
var content embed.FS

func HasContent() bool {
	files, err := content.ReadDir("app")
	if err != nil {
		return false
	}
	return len(files) > 1
}

func Handler() http.Handler {
	appFs, err := fs.Sub(content, "app")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/", handle404(http.FS(appFs)))
}

func handle404(root http.FileSystem) http.Handler {
	fileServer := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f, err := root.Open(r.URL.Path)
		if err != nil {
			if os.IsNotExist(err) {
				r.URL.Path = "/"
			}
		} else {
			_ = f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
}
