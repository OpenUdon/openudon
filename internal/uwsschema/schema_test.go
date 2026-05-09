package uwsschema

import (
	"os"
	"testing"
)

func TestPathForVersionFindsReadableSchema(t *testing.T) {
	path := PathForVersion(t.TempDir(), "1.0.0")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("schema path %s is not readable: %v", path, err)
	}
}

func TestModuleCacheSchemaPathFindsDependencySchema(t *testing.T) {
	if _, ok := uwsModuleVersion(); !ok {
		t.Skip("uws module version is unavailable, likely because the dependency is workspace-replaced")
	}
	path, ok := moduleCacheSchemaPath("1.0.0.json")
	if !ok {
		t.Fatalf("module cache schema path not found")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("module cache schema path %s is not readable: %v", path, err)
	}
}
