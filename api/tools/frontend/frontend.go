package frontend

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed app/*
var content embed.FS

func Handler() http.Handler {
	appFs, err := fs.Sub(content, "app")
	if err != nil {
		panic(err)
	}
	return http.StripPrefix("/", http.FileServer(http.FS(appFs)))
}
