package buildinfo

import (
	"reflect"
	"runtime/debug"
	"testing"
)

func TestParseUsesModuleVersionAndStableTags(t *testing.T) {
	info := Parse("devel", &debug.BuildInfo{
		GoVersion: "go1.25.0", Main: debug.Module{Path: Module, Version: "v0.2.0"},
		Settings: []debug.BuildSetting{{Key: "-tags", Value: "sqlite,release"}, {Key: "vcs.revision", Value: "abc"}, {Key: "vcs.modified", Value: "false"}},
	})
	if info.Version != "0.2.0" || info.MainPath != Module || info.Revision != "abc" {
		t.Fatalf("Parse() = %#v", info)
	}
	if !reflect.DeepEqual(info.BuildTags, []string{"release", "sqlite"}) {
		t.Fatalf("tags = %#v", info.BuildTags)
	}
}

func TestParseKeepsInjectedVersion(t *testing.T) {
	info := Parse("0.2.0-rc.1", &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}})
	if info.Version != "0.2.0-rc.1" || info.Module != Module {
		t.Fatalf("Parse() = %#v", info)
	}
}
