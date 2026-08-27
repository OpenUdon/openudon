package synthesize

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	browsertrust "github.com/OpenUdon/browsertools/contenttrust"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	uwstrust "github.com/OpenUdon/uws/contenttrust"
	"github.com/OpenUdon/uws/uws1"
)

// analyzePackageContentTrust is OpenUdon's explicit advisory analysis entry
// point. It is deliberately separate from execution-profile validation and
// returns an empty report for declaration-free legacy packages.
func analyzePackageContentTrust(ctx context.Context, exampleDir string, doc *uws1.Document) (*uwstrust.Report, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if doc == nil {
		return nil, fmt.Errorf("content-trust analysis requires a UWS document")
	}
	if doc.ContentTrust == nil {
		return &uwstrust.Report{}, nil
	}
	resolver, hasBrowserSources := packageBrowserContentTrustResolver(ctx, exampleDir, doc)
	if hasBrowserSources {
		return uwstrust.Analyze(ctx, doc, resolver)
	}
	return uwstrust.Analyze(ctx, doc)
}

type packageBrowserResolver struct {
	resolver *browsertrust.Resolver
	browser  map[string]bool
	invalid  map[string]bool
}

func (r *packageBrowserResolver) ResolveOperation(ctx context.Context, doc *uws1.Document, operation *uws1.Operation) (bool, uwstrust.OperationContract, error) {
	if err := ctx.Err(); err != nil {
		return false, uwstrust.OperationContract{}, err
	}
	if operation == nil || !r.browser[strings.TrimSpace(operation.SourceDescription)] {
		return false, uwstrust.OperationContract{}, nil
	}
	if r.invalid[strings.TrimSpace(operation.SourceDescription)] || r.resolver == nil {
		return true, uwstrust.OperationContract{}, fmt.Errorf("browser profile contract is unavailable")
	}
	return r.resolver.ResolveOperation(ctx, doc, operation)
}

func packageBrowserContentTrustResolver(ctx context.Context, exampleDir string, doc *uws1.Document) (*packageBrowserResolver, bool) {
	result := &packageBrowserResolver{browser: map[string]bool{}, invalid: map[string]bool{}}
	paths, err := packageartifacts.CollectBrowserProfilePaths(exampleDir)
	if err != nil {
		markBrowserContentTrustSources(doc, result.browser, result.invalid)
		return result, len(result.browser) > 0
	}
	allowed := make(map[string]bool, len(paths))
	for _, path := range paths {
		allowed[filepath.ToSlash(path)] = true
	}
	profiles := map[string]*profile.Profile{}
	for _, source := range doc.SourceDescriptions {
		if source == nil || source.Type != uws1.SourceDescriptionTypeBrowserProfile {
			continue
		}
		name := strings.TrimSpace(source.Name)
		result.browser[name] = true
		if err := ctx.Err(); err != nil {
			result.invalid[name] = true
			continue
		}
		relative, cleanErr := packageartifacts.CleanRelativePath(source.URL)
		if cleanErr != nil || !allowed[relative] {
			result.invalid[name] = true
			continue
		}
		path := filepath.Join(exampleDir, filepath.FromSlash(relative))
		data, _, readErr := evidencefile.ReadRegular(path, evidencefile.DefaultMaxBytes)
		if readErr != nil {
			result.invalid[name] = true
			continue
		}
		value, loadErr := parseBrowserContentTrustProfile(relative, data)
		if loadErr != nil {
			result.invalid[name] = true
			continue
		}
		profiles[name] = value
	}
	if len(result.browser) == 0 {
		return result, false
	}
	resolver, err := browsertrust.NewResolver(profiles)
	if err != nil {
		for name := range result.browser {
			result.invalid[name] = true
		}
		return result, true
	}
	result.resolver = resolver
	return result, true
}

func parseBrowserContentTrustProfile(path string, data []byte) (*profile.Profile, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return profile.ParseJSON(data)
	case ".yaml", ".yml":
		return profile.ParseYAML(data)
	default:
		return nil, fmt.Errorf("unsupported browser profile extension")
	}
}

func markBrowserContentTrustSources(doc *uws1.Document, browser, invalid map[string]bool) {
	if doc == nil {
		return
	}
	for _, source := range doc.SourceDescriptions {
		if source == nil || source.Type != uws1.SourceDescriptionTypeBrowserProfile {
			continue
		}
		name := strings.TrimSpace(source.Name)
		browser[name] = true
		invalid[name] = true
	}
}
