// Package buildinfo extracts stable local build metadata without coupling CLI
// flag handling to runtime/debug's wire representation.
package buildinfo

import (
	"runtime/debug"
	"sort"
	"strings"
)

const Module = "github.com/OpenUdon/openudon"

type Info struct {
	Version   string            `json:"version"`
	Module    string            `json:"module"`
	MainPath  string            `json:"main_path,omitempty"`
	GoVersion string            `json:"go_version,omitempty"`
	Revision  string            `json:"revision,omitempty"`
	BuildTags []string          `json:"build_tags,omitempty"`
	Settings  map[string]string `json:"settings,omitempty"`
}

func Current(linkedVersion string) Info {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return Parse(linkedVersion, nil)
	}
	return Parse(linkedVersion, build)
}

func Parse(linkedVersion string, build *debug.BuildInfo) Info {
	info := Info{Version: strings.TrimSpace(linkedVersion), Module: Module}
	if build != nil {
		info.GoVersion = build.GoVersion
		info.MainPath = strings.TrimSpace(build.Main.Path)
		if (info.Version == "" || info.Version == "devel") && build.Main.Version != "" && build.Main.Version != "(devel)" {
			info.Version = strings.TrimPrefix(build.Main.Version, "v")
		}
		settings := map[string]string{}
		for _, setting := range build.Settings {
			switch setting.Key {
			case "vcs.revision":
				info.Revision = setting.Value
			case "-tags":
				info.BuildTags = SplitTags(setting.Value)
			case "vcs.modified", "vcs.time", "vcs":
				settings[setting.Key] = setting.Value
			}
		}
		if len(settings) > 0 {
			info.Settings = settings
		}
	}
	if info.Version == "" {
		info.Version = "devel"
	}
	return info
}

func SplitTags(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == ' ' })
	tags := make([]string, 0, len(fields))
	for _, field := range fields {
		if field = strings.TrimSpace(field); field != "" {
			tags = append(tags, field)
		}
	}
	sort.Strings(tags)
	return tags
}
