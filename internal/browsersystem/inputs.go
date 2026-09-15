package browsersystem

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Inventory the union of actual build configurations, not their combined tags:
// a default-only file can import dependencies absent in either browser build.
func addInputGoDependencies(ctx context.Context, root, udon string, qualification bool, addFile func(string) error) error {
	seen := map[string]bool{}
	for _, repo := range []string{root, udon, filepath.Join(filepath.Dir(root), "browsertools")} {
		configurations := [][]string{nil}
		if qualification && repo == root {
			configurations = append(configurations, []string{"-tags=icot_ui_browser"}, []string{"-tags=browser_system_qualification"})
		}
		for _, flags := range configurations {
			args := append([]string{"go", "list", "-deps", "-test", "-json"}, flags...)
			data, err := command(ctx, repo, append(args, "./..."), nil)
			if err != nil {
				return errors.New("development_go_dependencies")
			}
			decoder := json.NewDecoder(bytes.NewReader(data))
			for {
				var pkg struct {
					Dir                                                                        string
					GoFiles, CgoFiles, CFiles, CXXFiles, HFiles, SFiles, SysoFiles, EmbedFiles []string
				}
				err := decoder.Decode(&pkg)
				if err == io.EOF {
					break
				}
				if err != nil {
					return errors.New("development_go_dependencies")
				}
				for _, files := range [][]string{pkg.GoFiles, pkg.CgoFiles, pkg.CFiles, pkg.CXXFiles, pkg.HFiles, pkg.SFiles, pkg.SysoFiles, pkg.EmbedFiles} {
					for _, name := range files {
						path := name
						if !filepath.IsAbs(path) {
							path = filepath.Join(pkg.Dir, name)
						}
						if !seen[path] {
							if addFile(path) != nil {
								return errors.New("development_go_dependencies")
							}
							seen[path] = true
						}
					}
				}
			}
		}
	}
	return nil
}

const InputVersion = "openudon.browser-qualification-input.v1"

// InputIdentity is a browser-free inventory, not evidence that tests executed.
// Its hash includes exact source locations: moving a prepared checkout requires
// a fresh baseline. No environment values or dependency paths leave this API.
type InputIdentity struct {
	Version string `json:"version"`
	SHA256  string `json:"sha256"`
}

func QualificationInput(ctx context.Context, root, udon string) (InputIdentity, error) {
	bad := errors.New("qualification_input")
	for _, path := range []string{root, udon} {
		if !filepath.IsAbs(path) || filepath.Clean(path) != path {
			return InputIdentity{}, bad
		}
		real, err := filepath.EvalSymlinks(path)
		if err != nil || real != path {
			return InputIdentity{}, bad
		}
	}
	before, err := qualificationHostIdentity()
	if err != nil || ctx.Err() != nil {
		return InputIdentity{}, bad
	}
	input, err := inputInventory(ctx, root, udon, true)
	if err != nil || ctx.Err() != nil {
		return InputIdentity{}, bad
	}
	after, err := qualificationHostIdentity()
	if err != nil || before != after {
		return InputIdentity{}, bad
	}
	return InputIdentity{Version: InputVersion, SHA256: hash([]byte(InputVersion + "\x00" + input + "\x00" + before))}, nil
}

func qualificationHostIdentity() (string, error) {
	data, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return "", err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		key, value, ok := strings.Cut(line, ":")
		if ok && (key == "Umask" || key == "Uid" || key == "Gid" || key == "Groups" || key == "NoNewPrivs" || key == "Seccomp") {
			values[key] = strings.Join(strings.Fields(value), " ")
		}
	}
	if len(values) != 6 || values["Umask"] != "0022" {
		return "", errors.New("qualification_host")
	}
	// Reboots and changes to user-namespace policy require a fresh baseline.
	for _, path := range []string{"/proc/sys/kernel/random/boot_id", "/etc/machine-id", "/proc/sys/user/max_user_namespaces", "/proc/sys/kernel/unprivileged_userns_clone", "/proc/sys/kernel/apparmor_restrict_unprivileged_userns"} {
		data, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			values[path] = "missing"
		} else if err != nil {
			return "", err
		} else {
			values[path] = hash(data)
		}
	}
	if authority := os.Getenv("XAUTHORITY"); authority != "" {
		data, err := os.ReadFile(authority)
		if err != nil {
			return "", err
		}
		values["display_authority"] = hash(data)
	}
	data, _ = json.Marshal(values)
	return hash(data), nil
}
