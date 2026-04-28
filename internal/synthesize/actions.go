package synthesize

import "github.com/tabilet/uws/uws1"

func cloneCriteria(items []*uws1.Criterion) []*uws1.Criterion {
	if len(items) == 0 {
		return nil
	}
	out := make([]*uws1.Criterion, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		clone := *item
		if len(item.Extensions) > 0 {
			clone.Extensions = cloneMap(item.Extensions)
		}
		out = append(out, &clone)
	}
	return out
}

func cloneFailureActions(items []*uws1.FailureAction) []*uws1.FailureAction {
	if len(items) == 0 {
		return nil
	}
	out := make([]*uws1.FailureAction, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		clone := *item
		clone.Criteria = cloneCriteria(item.Criteria)
		if len(item.Extensions) > 0 {
			clone.Extensions = cloneMap(item.Extensions)
		}
		out = append(out, &clone)
	}
	return out
}

func cloneSuccessActions(items []*uws1.SuccessAction) []*uws1.SuccessAction {
	if len(items) == 0 {
		return nil
	}
	out := make([]*uws1.SuccessAction, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		clone := *item
		clone.Criteria = cloneCriteria(item.Criteria)
		if len(item.Extensions) > 0 {
			clone.Extensions = cloneMap(item.Extensions)
		}
		out = append(out, &clone)
	}
	return out
}

func cloneMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}
