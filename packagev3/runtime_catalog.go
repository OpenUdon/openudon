package packagev3

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/OpenUdon/apitools"
	"github.com/OpenUdon/browsertools"
	"github.com/OpenUdon/openudon/wire"
	"github.com/OpenUdon/uws/binding"
)

const RuntimeCatalogVersion = "udon.function-catalog.v1"
const RuntimeSourceID = "udon-runtime-functions"
const RuntimeSourceKind = "runtime-function"
const MaxAPISources = 32

// RuntimeVerifier is a trusted consumer-owned adapter to its implementing
// runtime's independent catalog reproduction. It must check exact source bytes,
// complete table claims and revision against the actual qualified runtime.
// Producer assertions are never adapters; OpenUdon imports no private runtime.
// Errors are reduced to ErrPackage; caller cancellation remains identifiable.
type RuntimeVerifier func(context.Context, string, []byte, binding.ShapeTable) error

func runtimeRevision(data []byte) (string, error) {
	var catalog struct {
		Version   string            `json:"version"`
		Revision  string            `json:"revision"`
		Functions []json.RawMessage `json:"functions"`
	}
	if len(data) > 512<<10 || wire.DecodeStrictNumbers(data, &catalog) != nil || catalog.Version != RuntimeCatalogVersion || len(catalog.Revision) != 40 || strings.ToLower(catalog.Revision) != catalog.Revision || catalog.Functions == nil || len(catalog.Functions) > 128 {
		return "", ErrPackage
	}
	if _, err := hex.DecodeString(catalog.Revision); err != nil {
		return "", ErrPackage
	}
	return catalog.Revision, nil
}

func verifyRuntimeTable(ctx context.Context, source Source, data []byte, table binding.ShapeTable, verifier RuntimeVerifier) error {
	revision, err := runtimeRevision(data)
	if err != nil || verifier == nil || source.ID != RuntimeSourceID || len(table.Sources) != 1 || table.Sources[0] != (binding.Source{ID: source.ID, Kind: RuntimeSourceKind, SHA256: source.Artifact.SHA256}) || len(table.Operations) == 0 || len(table.Operations) > 128 || table.Validate() != nil {
		return ErrPackage
	}
	for _, op := range table.Operations {
		if op.Protocol != "fnct" || op.Selector.Kind != "id" || op.Selector.Value != op.Selector.Key || len(op.Aliases) != 0 || !op.Security.Known || len(op.Security.Alternatives) != 1 || len(op.Security.Alternatives[0].Requirements) != 0 {
			return ErrPackage
		}
	}
	// Give the trusted adapter independent input values, so its mutation cannot
	// change the retained snapshot or produce a different verified table.
	encoded, err := table.Marshal()
	if err != nil {
		return ErrPackage
	}
	copy, err := binding.ParseTable(encoded)
	if err != nil {
		return ErrPackage
	}
	if verifier(ctx, revision, append([]byte(nil), data...), copy) != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrPackage
	}
	return ctx.Err()
}

func sourceOptions(sources []Source, files map[string][]byte) (apitools.OperationShapeOptions, []Source) {
	options := apitools.OperationShapeOptions{MaxBytes: MaxFileBytes}
	runtime := []Source{}
	for _, source := range sources {
		if source.Kind == "browser-profile" {
			continue
		}
		if source.Kind == RuntimeSourceKind {
			runtime = append(runtime, source)
			continue
		}
		options.Sources = append(options.Sources, apitools.ShapeSourceInput{ID: source.ID, OperationSourceInput: apitools.OperationSourceInput{Kind: apitools.OperationSourceKind(source.Kind), Content: files[source.Artifact.Path]}})
	}
	return options, runtime
}

func buildShapes(ctx context.Context, sources []Source, files map[string][]byte, claims map[string][]byte, verifier RuntimeVerifier) (binding.ShapeTable, error) {
	options, runtime := sourceOptions(sources, files)
	table := binding.ShapeTable{Version: ShapeVersion, Sources: []binding.Source{}, Operations: []binding.OperationShape{}}
	if len(options.Sources) > 0 {
		var err error
		table, err = apitools.BuildOperationShapeTable(ctx, options)
		if err != nil {
			return binding.ShapeTable{}, ErrPackage
		}
		if apitools.VerifyOperationShapeTable(ctx, options, table) != nil {
			return binding.ShapeTable{}, ErrPackage
		}
	}
	browserOptions := browserShapeOptions(sources, files)
	if len(browserOptions.Sources) > 0 {
		browser, err := browsertools.BuildBrowserShapeTable(ctx, browserOptions)
		if err != nil || browsertools.VerifyBrowserShapeTable(ctx, browserOptions, browser) != nil {
			return binding.ShapeTable{}, ErrPackage
		}
		table.Sources = append(table.Sources, browser.Sources...)
		table.Operations = append(table.Operations, browser.Operations...)
	}
	for _, source := range runtime {
		claimed, err := binding.ParseTable(claims[source.ID])
		if err != nil || verifyRuntimeTable(ctx, source, files[source.Artifact.Path], claimed, verifier) != nil {
			if ctx.Err() != nil {
				return binding.ShapeTable{}, ctx.Err()
			}
			return binding.ShapeTable{}, ErrPackage
		}
		table.Sources = append(table.Sources, claimed.Sources...)
		table.Operations = append(table.Operations, claimed.Operations...)
	}
	return table, nil
}

func verifyShapes(ctx context.Context, sources []Source, files map[string][]byte, table binding.ShapeTable, verifier RuntimeVerifier) error {
	options, runtime := sourceOptions(sources, files)
	api := binding.ShapeTable{Version: ShapeVersion, Sources: []binding.Source{}, Operations: []binding.OperationShape{}}
	browser := binding.ShapeTable{Version: ShapeVersion, Sources: []binding.Source{}, Operations: []binding.OperationShape{}}
	functions := binding.ShapeTable{Version: ShapeVersion, Sources: []binding.Source{}, Operations: []binding.OperationShape{}}
	for _, s := range table.Sources {
		if s.Kind == "browser-profile" {
			browser.Sources = append(browser.Sources, s)
		} else if s.Kind == RuntimeSourceKind {
			functions.Sources = append(functions.Sources, s)
		} else {
			api.Sources = append(api.Sources, s)
		}
	}
	for _, op := range table.Operations {
		if op.Source.Kind == "browser-profile" {
			browser.Operations = append(browser.Operations, op)
		} else if op.Source.Kind == RuntimeSourceKind {
			functions.Operations = append(functions.Operations, op)
		} else {
			api.Operations = append(api.Operations, op)
		}
	}
	if len(options.Sources) > 0 {
		if apitools.VerifyOperationShapeTable(ctx, options, api) != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return ErrPackage
		}
	} else if len(api.Sources)+len(api.Operations) != 0 {
		return ErrPackage
	}
	browserOptions := browserShapeOptions(sources, files)
	if len(browserOptions.Sources) > 0 {
		if browsertools.VerifyBrowserShapeTable(ctx, browserOptions, browser) != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return ErrPackage
		}
	} else if len(browser.Sources)+len(browser.Operations) != 0 {
		return ErrPackage
	}
	if len(runtime) == 0 {
		if len(functions.Sources)+len(functions.Operations) != 0 {
			return ErrPackage
		}
		return nil
	}
	if len(runtime) != 1 || verifyRuntimeTable(ctx, runtime[0], files[runtime[0].Artifact.Path], functions, verifier) != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrPackage
	}
	return nil
}

func browserShapeOptions(sources []Source, files map[string][]byte) browsertools.BrowserShapeOptions {
	options := browsertools.BrowserShapeOptions{}
	for _, s := range sources {
		if s.Kind == "browser-profile" {
			options.Sources = append(options.Sources, browsertools.BrowserShapeSourceInput{ID: s.ID, Content: files[s.Artifact.Path]})
		}
	}
	return options
}
