// Package simulation adapts OpenUdon packages to the public UWS orchestrator
// and pure mock runtime. It has no network, credential resolver or executor.
package simulation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OpenUdon/uws/mockruntime"
	"github.com/OpenUdon/uws/uws1"
)

const Version = "openudon.simulate.v1"
const InputVersion = "openudon.simulate-input.v1"
const MaxResultBytes = 512 << 10
const MaxRequests = 128

type ResponseDefinition struct {
	Example json.RawMessage   `json:"example,omitempty"`
	Schema  *uws1.ParamSchema `json:"schema,omitempty"`
}

type Input struct {
	Version   string                        `json:"version"`
	Inputs    map[string]any                `json:"inputs,omitempty"`
	Responses map[string]ResponseDefinition `json:"responses,omitempty"`
}

type Options struct {
	RepoRoot, ExampleDir   string
	Inputs                 map[string]any
	Responses              map[string]ResponseDefinition
	Fixtures               *mockruntime.FixtureSet
	AllowGeneratedFallback bool
}

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Report struct {
	Version          string       `json:"version"`
	Tier             string       `json:"tier"`
	Status           string       `json:"status"`
	UWSVersion       string       `json:"uws_version,omitempty"`
	PackageSHA256    string       `json:"package_sha256,omitempty"`
	PackageUnchanged bool         `json:"package_unchanged"`
	Hypothetical     bool         `json:"hypothetical"`
	ValuesPolicy     string       `json:"values_policy"`
	Steps            []StepResult `json:"steps"`
	WouldBeRequests  []Request    `json:"would_be_requests"`
	Diagnostics      []Diagnostic `json:"diagnostics"`
}

type StepResult struct {
	StepID             string          `json:"step_id"`
	Binding            string          `json:"binding"`
	Effect             string          `json:"effect"`
	Outcome            string          `json:"simulation_outcome"`
	ResponseProvenance []string        `json:"response_provenance"`
	BrowserEvidence    string          `json:"browser_evidence,omitempty"`
	Outputs            json.RawMessage `json:"outputs,omitempty"`
}

type Request struct {
	StepID             string          `json:"step_id"`
	OperationID        string          `json:"operation_id,omitempty"`
	Effect             string          `json:"effect"`
	EndpointStatus     string          `json:"endpoint_status"`
	ResponseProvenance string          `json:"response_provenance"`
	WouldBeRequest     json.RawMessage `json:"would_be_request,omitempty"`
	Truncated          bool            `json:"truncated,omitempty"`
}

func Failed(code, message string) Report {
	return Report{Version: Version, Tier: "simulation", Status: "blocked", ValuesPolicy: "all preview scalar values and object field names redacted", Steps: []StepResult{}, WouldBeRequests: []Request{}, Diagnostics: []Diagnostic{{Code: code, Message: message}}}
}

func Run(ctx context.Context, options Options) Report {
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	before, err := capture(ctx, options.RepoRoot, options.ExampleDir)
	if err != nil {
		return Failed("package.invalid", "The review package is unavailable, inconsistent, unsafe or beyond simulation bounds.")
	}
	report := Report{Version: Version, Tier: "simulation", Status: "completed", UWSVersion: before.document.UWS, PackageSHA256: before.digest, ValuesPolicy: "all preview scalar values and object field names redacted", Steps: []StepResult{}, WouldBeRequests: []Request{}, Diagnostics: []Diagnostic{}}
	projection, err := project(before.document, options)
	if err != nil {
		report.Status = "blocked"
		report.Diagnostics = append(report.Diagnostics, Diagnostic{"projection.invalid", "The simulation projection cannot preserve this package within its supported bounds."})
		return finish(ctx, options, before, report)
	}
	report.Hypothetical = len(projection.pendingOperations) != 0
	mock, err := mockruntime.NewRuntime(projection.document, mockruntime.Options{Fixtures: projection.fixtures, AllowGeneratedFallback: options.AllowGeneratedFallback, ResponseResolver: mockruntime.ResponseResolverFunc(func(ctx context.Context, op *uws1.Operation) (mockruntime.ResponseDefinition, error) {
		definition, ok := projection.definitions[op.OperationID]
		if !ok {
			return mockruntime.ResponseDefinition{}, errors.New("an explicit fixture, response example or schema is required")
		}
		return mockruntime.ResponseDefinition{Example: definition.Example, Schema: definition.Schema}, nil
	})})
	if err != nil {
		report.Status = "blocked"
		report.Diagnostics = append(report.Diagnostics, Diagnostic{"fixtures.invalid", "The fixture set fails the public Mock Fixture Format 1.0 contract."})
		return finish(ctx, options, before, report)
	}
	observed := &observedRuntime{Runtime: mock, calls: map[string][]mockruntime.RequestRecord{}}
	projection.document.SetRuntime(observed)
	ctx = uws1.WithExecutionContext(ctx, &uws1.ExecutionContext{Inputs: options.Inputs})
	if err := projection.document.Execute(ctx); err != nil {
		report.Status = "blocked"
		report.Diagnostics = append(report.Diagnostics, Diagnostic{"simulation.incomplete", "The public mock runtime could not complete the projection; check fixtures, inputs, schemas, expression support and bounds. No real operation ran."})
	}
	records := projection.document.ExecutionRecords()
	for _, binding := range projection.steps {
		step := StepResult{StepID: binding.id, Binding: "bound", Effect: binding.effect, Outcome: "not_started", ResponseProvenance: []string{}}
		if binding.pending {
			step.Binding = "pending"
		}
		if binding.browser {
			step.BrowserEvidence = "mocked contract only; page state not verified"
		}
		seen := map[string]bool{}
		keys := make([]string, 0, len(records))
		for key := range records {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			record := records[key]
			if record.ID != binding.id || !strings.HasPrefix(record.Kind, "step:") {
				continue
			}
			switch record.Status {
			case "success":
				if step.Outcome != "failed" {
					step.Outcome = "simulated"
				}
			case "error", "running":
				step.Outcome = "failed"
			case "skipped":
				if step.Outcome == "not_started" {
					step.Outcome = "skipped"
				}
			}
			if record.Outputs != nil {
				step.Outputs, _ = redactedJSON(record.Outputs, 2048)
			}
		}
		callKeys := make([]string, 0, len(observed.calls))
		for key := range observed.calls {
			callKeys = append(callKeys, key)
		}
		sort.Strings(callKeys)
		for _, key := range callKeys {
			if !callBelongsToStep(key, binding.id, binding.operation) {
				continue
			}
			for _, call := range observed.calls[key] {
				provenance := string(call.ResponseKind)
				if !seen[provenance] {
					step.ResponseProvenance = append(step.ResponseProvenance, provenance)
					seen[provenance] = true
				}
				request := Request{StepID: binding.id, OperationID: call.OperationID, Effect: binding.effect, EndpointStatus: "bound", ResponseProvenance: provenance}
				if binding.pending {
					request.OperationID = ""
					request.EndpointStatus = "unresolved"
				} else {
					var raw any
					decoder := json.NewDecoder(strings.NewReader(string(call.Request)))
					decoder.UseNumber()
					if decoder.Decode(&raw) == nil {
						request.WouldBeRequest, request.Truncated = redactedJSON(raw, 4096)
					}
				}
				report.WouldBeRequests = append(report.WouldBeRequests, request)
			}
		}
		sort.Strings(step.ResponseProvenance)
		report.Steps = append(report.Steps, step)
	}
	return finish(ctx, options, before, report)
}

func finish(ctx context.Context, options Options, before packageSnapshot, report Report) Report {
	// A canceled computation still checks the package without granting action
	// authority; use an independent bounded read context for the final capture.
	verify, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	after, err := capture(verify, options.RepoRoot, options.ExampleDir)
	report.PackageUnchanged = err == nil && after.digest == before.digest
	if !report.PackageUnchanged {
		report.Status = "blocked"
		report.Diagnostics = append(report.Diagnostics, Diagnostic{"package.changed", "The package changed during simulation; discard this preview and inspect its revision."})
	}
	encoded, err := json.Marshal(report)
	if err != nil || len(encoded) > MaxResultBytes {
		result := Failed("result.bound", "The preview exceeded its result bound; no approval or execution authority was created.")
		result.PackageSHA256 = report.PackageSHA256
		result.PackageUnchanged = report.PackageUnchanged
		return result
	}
	return report
}

type observedRuntime struct {
	*mockruntime.Runtime
	mu    sync.Mutex
	calls map[string][]mockruntime.RequestRecord
}

func (runtime *observedRuntime) ExecuteLeaf(ctx context.Context, op *uws1.Operation) error {
	_, err := runtime.ExecuteLeafWithResult(ctx, op)
	return err
}

func (runtime *observedRuntime) ExecuteLeafWithResult(ctx context.Context, op *uws1.Operation) (any, error) {
	// Serialize only instantaneous mock leaf calls to associate the public
	// request observation with its public execution context. The public
	// orchestrator still owns scheduling, branching, retries and data flow.
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	previous := len(runtime.Runtime.RequestRecords())
	if previous >= MaxRequests {
		return nil, errors.New("simulation request bound reached")
	}
	value, err := runtime.Runtime.ExecuteLeafWithResult(ctx, op)
	requests := runtime.Runtime.RequestRecords()
	if len(requests) > previous {
		if state, ok := uws1.ExecutionContextFromContext(ctx); ok && state.Current != nil {
			runtime.calls[state.Current.Key] = append(runtime.calls[state.Current.Key], requests[previous])
		}
	}
	return value, err
}

func callBelongsToStep(key, step, operation string) bool {
	// Keys are owned by UWS; the literal suffix is its step-operation record
	// convention, also under workflow/iteration scopes. IDs forbid colons.
	return key == "stepop:"+step+":"+operation || strings.HasSuffix(key, "::stepop:"+step+":"+operation) || strings.HasPrefix(key, "stepop:"+step+":"+operation+"#iter:") || strings.Contains(key, "::stepop:"+step+":"+operation+"#iter:")
}

func redactedJSON(value any, limit int) (json.RawMessage, bool) {
	nodes := 0
	var redact func(any, int) any
	redact = func(value any, depth int) any {
		nodes++
		if nodes > 1024 || depth > 16 {
			return "[redacted]"
		}
		switch v := value.(type) {
		case map[string]any:
			result := map[string]any{}
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for index, key := range keys {
				result[fmt.Sprintf("[field %d]", index+1)] = redact(v[key], depth+1)
			}
			return result
		case []any:
			result := make([]any, 0, len(v))
			for _, child := range v {
				if len(result) >= 128 {
					result = append(result, "[redacted]")
					break
				}
				result = append(result, redact(child, depth+1))
			}
			return result
		default:
			return "[redacted]"
		}
	}
	encoded, err := json.Marshal(redact(value, 0))
	if err != nil || len(encoded) > limit {
		return json.RawMessage(`"[redacted: preview bound]"`), true
	}
	return encoded, nodes > 1024
}

// Request digests that include private values remain inside the public mock
// runtime. Only the existing package digest is exported as revision evidence.
