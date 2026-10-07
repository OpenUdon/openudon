package packagev3

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/OpenUdon/uws/schemas"
)

func TestEmbeddedCoreSchemasMatchExactSelectedUWS(t *testing.T) {
	archive, err := zip.NewReader(bytes.NewReader(schemaArchive), int64(len(schemaArchive)))
	if err != nil {
		t.Fatal(err)
	}
	if len(archive.File) != 17 {
		t.Fatal("published core inventory changed")
	}
	for _, file := range archive.File {
		reader, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		bytes, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}
		if err := reader.Close(); err != nil {
			t.Fatal(err)
		}
		current, err := os.ReadFile(schemas.PathForVersion(".", file.Name[:len(file.Name)-5]))
		if err != nil {
			t.Fatal(err)
		}
		if string(bytes) != string(current) {
			t.Fatalf("core schema %s differs from selected UWS", file.Name)
		}
	}
}
