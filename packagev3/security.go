package packagev3

import (
	"context"
	"sort"

	"github.com/OpenUdon/openudon/authority"
	"github.com/OpenUdon/openudon/credentialpolicy"
	"github.com/OpenUdon/uws/binding"
	"github.com/OpenUdon/uws/uws1"
)

func operationBinding(op *uws1.Operation, sources []Source) (binding.Binding, error) {
	var identity *Source
	for i := range sources {
		if sources[i].ID == op.SourceDescription {
			identity = &sources[i]
			break
		}
	}
	if identity == nil || identity.Kind == RuntimeSourceKind {
		return binding.Binding{}, ErrPackage
	}
	kind, value := "id", op.SourceOperationID
	if value == "" {
		value = op.OpenAPIOperationID
	}
	if op.SourceOperationRef != "" {
		kind, value = "ref", op.SourceOperationRef
	}
	if op.OpenAPIOperationRef != "" {
		kind, value = "ref", op.OpenAPIOperationRef
	}
	return binding.Binding{Source: binding.Source{ID: identity.ID, Kind: identity.Kind, SHA256: identity.Artifact.SHA256}, SelectorKind: kind, SelectorValue: value}, nil
}

func symbolicSecurity(security binding.Security) ([]binding.SecurityBinding, error) {
	if !security.Known {
		return []binding.SecurityBinding{}, nil
	}
	byName := map[string]binding.SecurityBinding{}
	for _, alternative := range security.Alternatives {
		for _, requirement := range alternative.Requirements {
			if credentialpolicy.ContainsLikelyValue([]byte(requirement.Scheme)) {
				return nil, ErrPackage
			}
			existing := byName[requirement.Scheme]
			existing.Scheme, existing.CredentialSlot = requirement.Scheme, requirement.Scheme
			for _, scope := range requirement.Scopes {
				found := false
				for _, old := range existing.Scopes {
					found = found || old == scope
				}
				if !found {
					existing.Scopes = append(existing.Scopes, scope)
				}
			}
			sort.Strings(existing.Scopes)
			byName[requirement.Scheme] = existing
		}
	}
	result := make([]binding.SecurityBinding, 0, len(byName))
	for _, item := range byName {
		result = append(result, item)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Scheme < result[j].Scheme })
	return result, nil
}

func declaredCredentials(ctx context.Context, doc *uws1.Document, sources []Source, table binding.ShapeTable) ([]string, error) {
	resolver, err := binding.NewResolver(table)
	if err != nil {
		return nil, ErrPackage
	}
	names := map[string]bool{}
	for _, op := range doc.Operations {
		if !op.HasSourceBinding() {
			continue
		}
		b, err := browserOperationBinding(op, sources)
		if err != nil {
			continue
		}
		result, err := resolver.Resolve(ctx, b)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			return nil, ErrPackage
		}
		if result.Status != binding.Resolved || result.Shape == nil {
			continue
		}
		symbolic, err := symbolicSecurity(result.Shape.Security)
		if err != nil {
			return nil, err
		}
		for _, slot := range symbolic {
			// Unsupported names remain in exact source/shape review metadata;
			// the unchanged handoff wire inventories only addressable slots.
			if authority.Identifier(slot.CredentialSlot) {
				names[slot.CredentialSlot] = true
			}
		}
		if len(names) > 64 {
			return nil, ErrPackage
		}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}
