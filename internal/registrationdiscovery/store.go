package registrationdiscovery

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// Store opens only an existing canonical owner-only private root, outside Git
// and the example. Each example has an isolated append-only revision chain.
// An interrupted write or abandoned lock fails closed; no history is reset.
type Store struct{ root *os.Root }

func Open(privateRoot, example string) (*Store, error) {
	if !filepath.IsAbs(privateRoot) || !filepath.IsAbs(example) || filepath.Clean(privateRoot) != privateRoot {
		return nil, ErrStorage
	}
	canonical, err := filepath.EvalSymlinks(privateRoot)
	if err != nil || canonical != privateRoot {
		return nil, ErrStorage
	}
	if resolved, err := filepath.EvalSymlinks(example); err == nil {
		example = resolved
	}
	if rel, err := filepath.Rel(example, privateRoot); err != nil || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return nil, ErrStorage
	}
	for parent := privateRoot; ; parent = filepath.Dir(parent) {
		if _, err := os.Lstat(filepath.Join(parent, ".git")); !os.IsNotExist(err) {
			return nil, ErrStorage
		}
		if filepath.Dir(parent) == parent {
			break
		}
	}
	root, err := os.OpenRoot(privateRoot)
	if err != nil {
		return nil, ErrStorage
	}
	defer root.Close()
	if info, err := root.Stat("."); err != nil || !info.IsDir() || info.Mode().Perm() != 0700 {
		return nil, ErrStorage
	}
	sum := sha256.Sum256([]byte(filepath.Clean(example)))
	name := fmt.Sprintf("registration-discovery-%x", sum[:])
	if err := root.Mkdir(name, 0700); err != nil && !os.IsExist(err) {
		return nil, ErrStorage
	}
	before, err := root.Lstat(name)
	if err != nil || !before.IsDir() || before.Mode().Perm() != 0700 {
		return nil, ErrStorage
	}
	child, err := root.OpenRoot(name)
	if err != nil {
		return nil, ErrStorage
	}
	after, err := child.Stat(".")
	if err != nil || !os.SameFile(before, after) {
		child.Close()
		return nil, ErrStorage
	}
	return &Store{root: child}, nil
}

func (s *Store) Close() error { return s.root.Close() }

func (s *Store) Read() (Inventory, error) {
	current := empty()
	dir, err := s.root.Open(".")
	if err != nil {
		return Inventory{}, ErrStorage
	}
	entries, err := dir.ReadDir(515)
	dir.Close()
	if err != nil && err != io.EOF {
		return Inventory{}, ErrStorage
	}
	if len(entries) >= 515 {
		return Inventory{}, ErrStorage
	}
	names := []string{}
	for _, entry := range entries {
		if entry.Name() == "write-lock" {
			continue
		}
		// A leftover staging file or unexpected entry needs explicit recovery.
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	for index, name := range names {
		seq := index + 1
		if seq > 512 || name != fmt.Sprintf("%06d.json", seq) {
			return Inventory{}, ErrStorage
		}
		before, err := s.root.Lstat(name)
		if err != nil || !before.Mode().IsRegular() || before.Mode().Perm() != 0600 || before.Size() > 128<<10 {
			return Inventory{}, ErrStorage
		}
		f, err := s.root.Open(name)
		if err != nil {
			return Inventory{}, ErrStorage
		}
		data, readErr := io.ReadAll(io.LimitReader(f, (128<<10)+1))
		after, statErr := f.Stat()
		f.Close()
		var next Inventory
		if readErr != nil || statErr != nil || !os.SameFile(before, after) || int64(len(data)) != after.Size() || len(data) > 128<<10 || evidencefile.DecodeStrict(data, &next) != nil || !valid(next) || next.Sequence != seq || next.PreviousRevision != current.Revision || next.Revision != digest(next) {
			return Inventory{}, ErrStorage
		}
		current = next
	}
	return current, nil
}

func (s *Store) Change(c Change, observedURL string) (Inventory, error) {
	// Cross-process exclusion prevents two UIs from overwriting one revision.
	if err := s.root.Mkdir("write-lock", 0700); err != nil {
		return Inventory{}, ErrStorage
	}
	defer s.root.Remove("write-lock")
	i, err := s.Read()
	if err != nil {
		return Inventory{}, err
	}
	i, err = apply(i, c, observedURL)
	if err != nil {
		return Inventory{}, err
	}
	data, err := json.Marshal(i)
	if err != nil || len(data) > 128<<10 {
		return Inventory{}, ErrInvalid
	}
	// A fixed staging name is protected by the lock. Never truncate a leftover
	// staging file: an interrupted write needs explicit local reconciliation.
	f, err := s.root.OpenFile("pending.json", os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return Inventory{}, ErrStorage
	}
	_, err = f.Write(data)
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil || closeErr != nil {
		return Inventory{}, ErrStorage
	}
	name := fmt.Sprintf("%06d.json", i.Sequence)
	if _, err := s.root.Lstat(name); !os.IsNotExist(err) {
		return Inventory{}, ErrStorage
	}
	if err := s.root.Rename("pending.json", name); err != nil {
		return Inventory{}, ErrStorage
	}
	dir, err := s.root.Open(".")
	if err != nil {
		return Inventory{}, ErrStorage
	}
	err = dir.Sync()
	dir.Close()
	if err != nil {
		return Inventory{}, ErrStorage
	}
	return i, nil
}
