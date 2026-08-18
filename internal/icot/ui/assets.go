package ui

import (
	"embed"
	"net/http"
)

//go:embed assets/index.html assets/app.js assets/style.css
var assetFiles embed.FS

func serveEmbedded(w http.ResponseWriter, name, contentType string) {
	data, err := assetFiles.ReadFile(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "embedded UI asset is unavailable")
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
