// Package catalogpolicy owns reusable provider catalog selection decisions.
package catalogpolicy

import (
	"fmt"
	"sort"
	"strings"

	"github.com/OpenUdon/apitools/catalog"
)

func SelectOpenAPIReference(providerKey, specRefID string) (catalog.SpecReference, catalog.Provider, error) {
	providerKey = strings.TrimSpace(providerKey)
	if providerKey == "" {
		return catalog.SpecReference{}, catalog.Provider{}, fmt.Errorf("missing --provider")
	}
	provider, ok := catalog.FindBuiltInProvider(providerKey)
	if !ok {
		return catalog.SpecReference{}, catalog.Provider{}, fmt.Errorf("unknown provider %q", providerKey)
	}
	var refs []catalog.SpecReference
	for _, ref := range provider.SpecReferences {
		if ref.Kind == catalog.SpecKindOpenAPI {
			refs = append(refs, ref)
		}
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	if specRefID = strings.TrimSpace(specRefID); specRefID != "" {
		for _, ref := range refs {
			if ref.ID == specRefID {
				return ref, provider, nil
			}
		}
		return catalog.SpecReference{}, provider, fmt.Errorf("provider %q has no OpenAPI spec reference %q", provider.ID, specRefID)
	}
	if len(refs) == 0 {
		return catalog.SpecReference{}, provider, fmt.Errorf("provider %q has no directly importable OpenAPI spec; inspect catalog metadata for Discovery, Smithy, Stone, human-docs, or user-provided OpenAPI guidance", provider.ID)
	}
	if len(refs) > 1 {
		ids := make([]string, 0, len(refs))
		for _, ref := range refs {
			ids = append(ids, ref.ID)
		}
		return catalog.SpecReference{}, provider, fmt.Errorf("provider %q has multiple OpenAPI specs; pass --spec (%s)", provider.ID, strings.Join(ids, ", "))
	}
	return refs[0], provider, nil
}
