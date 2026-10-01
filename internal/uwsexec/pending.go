package uwsexec

import "github.com/OpenUdon/uws/uws1"

// PendingStepIDs is an observation over every workflow and branch, including
// unreachable ones. Public UWS ValidateExecutable owns the admission decision.
func PendingStepIDs(doc *uws1.Document) []string {
	var ids []string
	var steps func([]*uws1.Step)
	steps = func(list []*uws1.Step) {
		for _, s := range list {
			if s == nil {
				continue
			}
			if s.Pending != nil {
				ids = append(ids, s.StepID)
			}
			steps(s.Steps)
			for _, c := range s.Cases {
				if c != nil {
					steps(c.Steps)
				}
			}
			steps(s.Default)
		}
	}
	if doc != nil {
		for _, w := range doc.Workflows {
			if w == nil {
				continue
			}
			steps(w.Steps)
			for _, c := range w.Cases {
				if c != nil {
					steps(c.Steps)
				}
			}
			steps(w.Default)
		}
	}
	return ids
}
