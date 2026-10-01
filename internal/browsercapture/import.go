package browsercapture

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/OpenUdon/openudon/internal/artifactwriter"
	"github.com/OpenUdon/openudon/internal/browserauthor"
	"github.com/OpenUdon/openudon/internal/browsertransaction"
	"github.com/OpenUdon/openudon/internal/elicitor"
	"github.com/OpenUdon/openudon/internal/evidencefile"
)

type authenticationOptions struct {
	continuation string
	complete     func(context.Context, browserauthor.Event) (*profileImport, error)
}

// profileImport holds immutable, independently reconstructed native sources.
// Preparing it is read-only. The shared transport alone can call commit after
// an issued exact import card is approved, after the worker has joined.
type profileImport struct {
	result Result
	commit func(context.Context) error
}

func (p *profileImport) validate(mode string, view View, command Command) error {
	if p == nil || view.State != "import_review" || view.Result == nil || *view.Result != p.result {
		return errInput
	}
	expected := Command{Authentication: &browserauthor.Response{Kind: "confirm", Confirmed: true}}
	if mode == Registration {
		expected = Command{Registration: &RegistrationCommand{Type: "finish", Confirmed: true}}
	}
	if !reflect.DeepEqual(command, expected) {
		return errInput
	}
	return nil
}

type importedFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}
type importReceipt struct {
	Version     string                         `json:"version"`
	StartSHA256 string                         `json:"start_sha256"`
	Transaction browsertransaction.Transaction `json:"transaction"`
	Effect      string                         `json:"effect"`
	Files       []importedFile                 `json:"files"`
}

func prepareProfileImport(ctx context.Context, root string, input elicitor.VirtualBrowserTransactionInput, files []artifactwriter.GeneratedFile, startSHA string, guard func(context.Context) error) (*profileImport, error) {
	if input.Transaction.State != browsertransaction.StateReviewed || guard == nil || !evidencefile.ValidSHA256(startSHA) {
		return nil, errors.New("reviewed native capture required")
	}
	canonical, err := browsertransaction.CanonicalBytes(input.Transaction)
	if err != nil {
		return nil, err
	}
	input.Transaction, err = browsertransaction.Decode(canonical)
	if err != nil {
		return nil, err
	}
	input.Sources = append([]elicitor.VirtualBrowserSourceInput(nil), input.Sources...)
	for i := range input.Sources {
		input.Sources[i].Source = append([]byte(nil), input.Sources[i].Source...)
		input.Sources[i].Review = append([]byte(nil), input.Sources[i].Review...)
	}
	if err := guard(ctx); err != nil {
		return nil, err
	}
	discovery, err := elicitor.DiscoverVirtualBrowserSources([]elicitor.VirtualBrowserTransactionInput{input}, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if files == nil {
		// Native materialization owns profile targets, canonical bytes and reviews.
		for _, plan := range discovery.Plans {
			path, err := artifactwriter.SafeExampleTarget(root, plan.TargetPath)
			if err != nil {
				return nil, err
			}
			files = append(files, artifactwriter.GeneratedFile{Path: path, Content: string(plan.MaterializedContent)})
			if plan.ReviewPath != "" {
				path, err := artifactwriter.SafeExampleTarget(root, plan.ReviewPath)
				if err != nil {
					return nil, err
				}
				files = append(files, artifactwriter.GeneratedFile{Path: path, Content: string(plan.MaterializedReview)})
			}
		}
	}
	if len(files) == 0 {
		return nil, errors.New("capture materialization missing")
	}
	files = append([]artifactwriter.GeneratedFile(nil), files...)
	digest, err := browsertransaction.Digest(input.Transaction)
	if err != nil {
		return nil, err
	}
	digest = strings.TrimPrefix(digest, "sha256:")
	// Conservatively classify captures that can describe login/submission as
	// write. Profile import grants no executor authority or production approval.
	result := Result{ProfileID: input.Transaction.ID, TransactionSHA256: digest, Effect: "write"}
	receipt := importReceipt{Version: "openudon.browser-capture-import.v1", StartSHA256: startSHA, Transaction: input.Transaction, Effect: result.Effect}
	for _, file := range files {
		relative, err := filepath.Rel(root, file.Path)
		if err != nil {
			return nil, err
		}
		safe, err := artifactwriter.SafeExampleTarget(root, relative)
		if err != nil || safe != file.Path || file.Remove {
			return nil, errors.New("capture target invalid")
		}
		sum := sha256.Sum256([]byte(file.Content))
		receipt.Files = append(receipt.Files, importedFile{Path: filepath.ToSlash(relative), SHA256: hex.EncodeToString(sum[:])})
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return nil, err
	}
	receiptPath, err := artifactwriter.SafeExampleTarget(root, filepath.Join("expected", "browser-capture", input.Transaction.ID+".json"))
	if err != nil {
		return nil, err
	}
	files = append(files, artifactwriter.GeneratedFile{Path: receiptPath, Content: string(data) + "\n"})
	return &profileImport{result: result, commit: func(ctx context.Context) error {
		check := func() error {
			if err := guard(ctx); err != nil {
				return err
			}
			_, err := elicitor.DiscoverVirtualBrowserSources([]elicitor.VirtualBrowserTransactionInput{input}, time.Now().UTC())
			return err
		}
		if err := check(); err != nil {
			return err
		}
		_, err := artifactwriter.CommitChecked(artifactwriter.Prepared{ExampleRoot: root, Files: files}, false, check)
		return err
	}}, nil
}
