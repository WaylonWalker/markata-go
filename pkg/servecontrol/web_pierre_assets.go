package servecontrol

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed pierre-diffs/*
var pierreDiffAssets embed.FS

func pierreDiffAssetsHandler() http.Handler {
	assets, err := fs.Sub(pierreDiffAssets, "pierre-diffs")
	if err != nil {
		panic(err) // The embedded directory is required at compile time.
	}
	const prefix = "/_markata/assets/pierre-diffs/"
	files := http.StripPrefix(prefix, http.FileServer(http.FS(assets)))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		files.ServeHTTP(w, r)
	})
}
