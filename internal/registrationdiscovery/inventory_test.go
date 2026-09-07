package registrationdiscovery

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func privateStore(t *testing.T) (*Store, string, string) {
	t.Helper()
	root, example := t.TempDir(), t.TempDir()
	if err := os.Chmod(root, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := Open(root, example)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s, root, example
}

func TestInventoryReviewCoverageAndPrivateHistory(t *testing.T) {
	s, root, example := privateStore(t)
	initial, err := s.Read()
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Change(Change{Revision: initial.Revision, Action: "add", ID: "advertiser", RegistrationType: "advertiser", URL: "https://app.example.test/register?action=startnew"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Change(Change{Revision: initial.Revision, Action: "review"}, ""); !errors.Is(err, ErrStale) {
		t.Fatalf("stale write: %v", err)
	}
	reviewed, err := s.Change(Change{Revision: first.Revision, Action: "review"}, "")
	if err != nil || reviewed.OwnerReview != "reviewed" || reviewed.Coverage != "unknown" {
		t.Fatalf("review changed coverage: %v", err)
	}
	selected, err := s.Change(Change{Revision: reviewed.Revision, Action: "select", ID: "advertiser"}, "")
	if err != nil || selected.OwnerReview != "pending" || selected.SelectedID != "advertiser" {
		t.Fatalf("selection: %v", err)
	}
	partial, err := s.Change(Change{Revision: selected.Revision, Action: "coverage", Coverage: "partial", Limitations: []string{}}, "")
	if err != nil || partial.OwnerReview != "pending" {
		t.Fatal(err)
	}
	second, err := Open(root, example)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	loaded, err := second.Read()
	if err != nil || loaded.Revision != partial.Revision || len(loaded.Candidates) != 1 {
		t.Fatalf("reload: %v", err)
	}
	entries, _ := os.ReadDir(root)
	files, _ := os.ReadDir(filepath.Join(root, entries[0].Name()))
	if len(files) != 4 {
		t.Fatalf("history lost: %d", len(files))
	}
	for _, f := range files {
		info, _ := f.Info()
		if info.Mode().Perm() != 0600 {
			t.Fatal("public history")
		}
	}
	other, err := Open(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	separate, err := other.Read()
	if err != nil || len(separate.Candidates) != 0 {
		t.Fatal("cross-workspace inventory")
	}
}

func TestInventoryRejectsFalseClaimsAndMalformedChanges(t *testing.T) {
	for _, change := range []Change{
		{Action: "coverage", Coverage: "complete", Limitations: []string{}},
		{Action: "coverage", Coverage: "owner_reviewed", Limitations: []string{}},
		{Action: "coverage", Coverage: "partial", Limitations: []string{"unknown_routes", "unknown_routes"}},
		{Action: "review", URL: "https://app.example.test"},
		{Action: "add", ID: "advertiser", RegistrationType: "advertiser", URL: "https://user:password@app.example.test/register"},
		{Action: "add", ID: "advertiser", RegistrationType: "advertiser", URL: "javascript:alert(1)"},
		{Action: "add_observed", ID: "advertiser", RegistrationType: "advertiser", URL: "https://app.example.test"},
		{Action: "add_observed", ID: "advertiser", RegistrationType: "advertiser"},
		{Action: "select", ID: "missing"},
	} {
		i := empty()
		change.Revision = i.Revision
		if _, err := apply(i, change, ""); !errors.Is(err, ErrInvalid) {
			t.Fatalf("accepted %s: %v", change.Action, err)
		}
	}
	i := empty()
	observed, err := apply(i, Change{Revision: i.Revision, Action: "add_observed", ID: "publisher", RegistrationType: "publisher"}, "https://app.example.test/publisher")
	if err != nil || observed.Candidates[0].Source != "browser_observed" || observed.Coverage != "unknown" {
		t.Fatalf("observation: %v", err)
	}
}

func TestInventoryStorageRejectsUnsafeRootsAndDamagedHistory(t *testing.T) {
	s, root, example := privateStore(t)
	if err := os.Chmod(root, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root, example); err == nil {
		t.Fatal("public root accepted")
	}
	os.Chmod(root, 0700)
	if _, err := Open(root, root); err == nil {
		t.Fatal("example accepted as private root")
	}
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(alias, example); err == nil {
		t.Fatal("symlink root accepted")
	}
	if err := os.Mkdir(filepath.Join(root, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(root, example); err == nil {
		t.Fatal("Git root accepted")
	}
	os.Remove(filepath.Join(root, ".git"))
	i, _ := s.Read()
	if _, err := s.Change(Change{Revision: i.Revision, Action: "review"}, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.root.Rename("000001.json", "000002.json"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatal("history gap accepted")
	}
	s.root.Rename("000002.json", "000001.json")
	if err := s.root.Chmod("000001.json", 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Read(); err == nil {
		t.Fatal("public history accepted")
	}
}

func TestInventoryConcurrentWritersCannotOverwriteHistory(t *testing.T) {
	s, root, example := privateStore(t)
	other, err := Open(root, example)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	i, _ := s.Read()
	start, results := make(chan struct{}), make(chan error, 2)
	for _, store := range []*Store{s, other} {
		go func(store *Store) {
			<-start
			_, err := store.Change(Change{Revision: i.Revision, Action: "review"}, "")
			results <- err
		}(store)
	}
	close(start)
	successes := 0
	for range 2 {
		if err := <-results; err == nil {
			successes++
		} else if !errors.Is(err, ErrStorage) && !errors.Is(err, ErrStale) {
			t.Fatal(err)
		}
	}
	current, err := s.Read()
	if err != nil || successes != 1 || current.Sequence != 1 {
		t.Fatalf("concurrent history: successes=%d sequence=%d err=%v", successes, current.Sequence, err)
	}
}

func TestInventoryOversizedEncodingDoesNotPoisonHistory(t *testing.T) {
	s, _, _ := privateStore(t)
	i, _ := s.Read()
	for _, id := range []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"} {
		next, err := s.Change(Change{Revision: i.Revision, Action: "add", ID: id, RegistrationType: "advertiser", URL: "https://app.example.test/register?q=" + strings.Repeat("<", 2000)}, "")
		if errors.Is(err, ErrInvalid) {
			if i.Sequence == 0 {
				t.Fatal("oversize fixture rejected before exercising encoded history limit")
			}
			loaded, err := s.Read()
			if err != nil || loaded.Revision != i.Revision {
				t.Fatal("oversize write poisoned previous history")
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		i = next
	}
	t.Fatal("encoded size limit was not enforced")
}
