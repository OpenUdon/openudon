package ui

import (
	"embed"
	"fmt"
	"net/http"
)

//go:embed assets/index.html assets/app.js assets/style.css
var assetFiles embed.FS

func serveEmbedded(w http.ResponseWriter, name, contentType string) error {
	data, err := assetFiles.ReadFile(name)
	if err != nil {
		return fmt.Errorf("read embedded UI asset %s: %w", name, err)
	}
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
	return nil
}
