package browsercheck

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

type Cache struct{ Root, Input string }
type cacheKey struct{}

func WithCache(ctx context.Context, cache *Cache) context.Context {
	return context.WithValue(ctx, cacheKey{}, cache)
}
func key(value string) string { h := sha256.Sum256([]byte(value)); return hex.EncodeToString(h[:]) }
func OpenCache(root, input string) (*Cache, error) {
	if !filepath.IsAbs(root) || filepath.Clean(root) != root || len(input) != 64 {
		return nil, errors.New("development_cache")
	}
	if _, err := hex.DecodeString(input); err != nil {
		return nil, errors.New("development_cache")
	}
	// Resolve the nearest existing parent before creating anything. A cache path
	// through a symlink must not create directories in a source workspace.
	ancestor := root
	for {
		_, err := os.Lstat(ancestor)
		if err == nil {
			real, e := filepath.EvalSymlinks(ancestor)
			if e != nil || real != ancestor {
				return nil, errors.New("development_cache")
			}
			break
		}
		if !os.IsNotExist(err) {
			return nil, errors.New("development_cache")
		}
		ancestor = filepath.Dir(ancestor)
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, errors.New("development_cache")
	}
	real, err := filepath.EvalSymlinks(root)
	info, statErr := os.Lstat(root)
	if err != nil || statErr != nil || real != root || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, errors.New("development_cache")
	}
	return &Cache{Root: root, Input: input}, nil
}

// TreeDigest includes file bytes/modes and relative symlink targets. Links must
// resolve inside the tree; cached build artifacts themselves disallow links.
func TreeDigest(root string, links bool) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		fmt.Fprintf(h, "%s\x00%o\x00", rel, info.Mode())
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			if !links {
				return errors.New("cache_symlink")
			}
			target, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			r, err := filepath.Rel(root, target)
			if err != nil || r == ".." || strings.HasPrefix(r, ".."+string(os.PathSeparator)) {
				return errors.New("cache_symlink")
			}
			value, err := os.Readlink(path)
			if err != nil {
				return err
			}
			io.WriteString(h, value)
		case info.IsDir():
			io.WriteString(h, "directory")
		case info.Mode().IsRegular():
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(h, f)
			closeErr := f.Close()
			if copyErr != nil || closeErr != nil {
				return errors.New("cache_read")
			}
		default:
			return errors.New("cache_file")
		}
		io.WriteString(h, "\x00")
		return nil
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyArtifact(from, to string) error {
	info, err := os.Lstat(from)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.Mkdir(to, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(from)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyArtifact(filepath.Join(from, e.Name()), filepath.Join(to, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	if !info.Mode().IsRegular() {
		return errors.New("cache_file")
	}
	in, err := os.Open(from)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	return errors.Join(err, out.Sync(), out.Close())
}

// Build caches only the explicit immutable output of a build callback. Runtime
// fixtures, forms, browser profiles and process state never enter this path.
// Without a development cache in context, qualification builds remain fresh.
func Build(ctx context.Context, name, target string, build func(string) error) (resultErr error) {
	finish := Span(ctx, "build_"+name)
	defer func() { finish(resultErr) }()
	c, _ := ctx.Value(cacheKey{}).(*Cache)
	if c == nil {
		return build(target)
	}
	if !c.validRoot() {
		return errors.New("development_cache")
	}
	input := key(c.Input + "\x00" + name)
	entry := filepath.Join(c.Root, "build-"+input)
	type manifest struct {
		Input  string `json:"input"`
		Digest string `json:"digest"`
	}
	restore := func() error {
		real, err := filepath.EvalSymlinks(entry)
		info, statErr := os.Lstat(entry)
		if err != nil || statErr != nil || real != entry || !info.IsDir() || info.Mode().Perm() != 0700 {
			return errors.New("cached_build_invalid")
		}
		data, info, err := evidencefile.ReadRegular(filepath.Join(entry, "manifest.json"), 4096)
		var m manifest
		if err != nil || info.Mode().Perm() != 0600 || evidencefile.DecodeStrict(data, &m) != nil || m.Input != input {
			return errors.New("cached_build_invalid")
		}
		artifact := filepath.Join(entry, "artifact")
		digest, err := TreeDigest(artifact, false)
		if err != nil || digest != m.Digest {
			return errors.New("cached_build_invalid")
		}
		if err := copyArtifact(artifact, target); err != nil {
			return err
		}
		copied, err := TreeDigest(target, false)
		if err != nil || copied != digest {
			return errors.New("cached_build_copy")
		}
		return nil
	}
	if _, err := os.Lstat(entry); err == nil {
		return restore()
	} else if !os.IsNotExist(err) {
		return err
	}
	temp, err := os.MkdirTemp(c.Root, ".build-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	artifact := filepath.Join(temp, "artifact")
	if err := build(artifact); err != nil {
		return err
	}
	digest, err := TreeDigest(artifact, false)
	if err != nil {
		return err
	}
	data, _ := json.Marshal(manifest{input, digest})
	if err := os.WriteFile(filepath.Join(temp, "manifest.json"), data, 0600); err != nil {
		return err
	}
	if err := os.Rename(temp, entry); err != nil {
		if _, e := os.Stat(entry); e != nil {
			return err
		}
	}
	return restore()
}

func (c *Cache) validRoot() bool {
	real, err := filepath.EvalSymlinks(c.Root)
	info, e := os.Lstat(c.Root)
	return err == nil && e == nil && real == c.Root && info.IsDir() && info.Mode().Perm() == 0700
}
func (c *Cache) ReadResult(stage string) ([]byte, error) {
	if !c.validRoot() {
		return nil, errors.New("development_cache")
	}
	data, info, err := evidencefile.ReadRegular(filepath.Join(c.Root, "result-"+key(c.Input+"\x00"+stage)+".json"), 16<<20)
	if err != nil {
		return nil, err
	}
	if info.Mode().Perm() != 0600 {
		return nil, errors.New("development_cache")
	}
	return data, nil
}
func (c *Cache) WriteResult(stage string, data []byte) error {
	if !c.validRoot() {
		return errors.New("development_cache")
	}
	f, err := os.CreateTemp(c.Root, ".result-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, err = f.Write(data)
	err = errors.Join(err, f.Sync(), f.Close())
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), filepath.Join(c.Root, "result-"+key(c.Input+"\x00"+stage)+".json"))
}
