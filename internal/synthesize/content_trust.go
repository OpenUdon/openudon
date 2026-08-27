package synthesize

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	browsertrust "github.com/OpenUdon/browsertools/contenttrust"
	"github.com/OpenUdon/browsertools/profile"
	"github.com/OpenUdon/openudon/internal/evidencefile"
	"github.com/OpenUdon/openudon/internal/packageartifacts"
	uwstrust "github.com/OpenUdon/uws/contenttrust"
	uwsconvert "github.com/OpenUdon/uws/convert"
	"github.com/OpenUdon/uws/uws1"
)

const contentTrustAnalysisUnavailableMessage = "operation resolver failed while describing the operation"

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

func loadPackageContentTrustReport(ctx context.Context, result Result) (*uwstrust.Report, bool, error) {
	data, _, err := evidencefile.ReadRegular(result.UWSPath, evidencefile.DefaultMaxBytes)
	if err != nil {
		return nil, false, err
	}
	var doc uws1.Document
	switch strings.ToLower(filepath.Ext(result.UWSPath)) {
	case ".json":
		err = uwsconvert.UnmarshalJSON(data, &doc)
	case ".hcl":
		err = uwsconvert.UnmarshalHCL(data, &doc)
	default:
		err = uwsconvert.UnmarshalYAML(data, &doc)
	}
	if err != nil {
		return nil, false, err
	}
	declared := doc.ContentTrust != nil
	report, err := analyzePackageContentTrust(ctx, result.ExampleDir, &doc)
	return report, declared, err
}

func assessContentTrust(ctx context.Context, quality *QualityReport, result Result) error {
	report, declared, err := loadPackageContentTrustReport(ctx, result)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		// Ordinary UWS assessment owns malformed/missing document failures. Keep
		// this pass advisory and never copy underlying error text into evidence.
		quality.add(uwstrust.CodeResolverFailure, "warn", contentTrustAnalysisUnavailableMessage, "severity=warning; path=contentTrust")
		return nil
	}
	if !declared {
		return nil
	}
	if len(report.Findings) == 0 {
		quality.add("content_trust.analysis", "pass", "content-trust analysis found no advisory issues", "")
		return nil
	}
	for _, finding := range report.Findings {
		quality.add(finding.Code, "warn", finding.Message, fmt.Sprintf("severity=%s; path=%s", finding.Severity, finding.Path))
	}
	return nil
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
