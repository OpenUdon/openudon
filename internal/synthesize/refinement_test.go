package synthesize

import (
	"context"
	"fmt"
	"testing"
)

func TestClassifyFailureClass(t *testing.T) {
	cases := []struct {
		name   string
		report *QualityReport
		err    error
		want   string
	}{
		{name: "validation", report: &QualityReport{Status: "fail"}, want: "validation"},
		{name: "model decode", err: fmt.Errorf("decode intent JSON: bad"), want: "model"},
		{name: "model context", err: context.DeadlineExceeded, want: "model"},
		{name: "validation intent", err: fmt.Errorf("generated intent referenced unavailable OpenAPI document"), want: "validation"},
		{name: "infra write", err: fmt.Errorf("write refinement report: denied"), want: "infra"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := classifyFailureClass(tc.report, tc.err); got != tc.want {
				t.Fatalf("class = %q, want %q", got, tc.want)
			}
		})
	}
}
