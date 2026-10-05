package brokerhandoff

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"

	"github.com/OpenUdon/openudon/internal/evidencefile"
)

// PrivateConfig is the external executor's private transport contract, not a
// portable run artifact. Capability never grants broader authority than the
// concrete reviewed Authority. The host broker must enforce that authority.
type PrivateConfig struct {
	Version      string              `json:"version"`
	SocketPath   string              `json:"socket_path"`
	Capability   string              `json:"capability"`
	RunID        string              `json:"run_id"`
	PolicyDigest string              `json:"policy_digest"`
	Invocations  []PrivateInvocation `json:"invocations"`
}

type PrivateInvocation struct {
	StepID       string    `json:"step_id"`
	OperationID  string    `json:"operation_id"`
	InvocationID string    `json:"invocation_id"`
	Bindings     []Binding `json:"bindings,omitempty"`
}

func (PrivateConfig) String() string   { return "[private broker config]" }
func (PrivateConfig) GoString() string { return "[private broker config]" }

// ReadPrivate refuses a public or replaced file and returns no transport values
// in its errors. Callers pass the private file to Udon, never serialize it into
// approval, run config, evidence or a credential environment variable.
func ReadPrivate(path string, a Authority) (PrivateConfig, error) {
	if !filepath.IsAbs(path) || filepath.Clean(path) != path {
		return PrivateConfig{}, errors.New("broker config must be an absolute private file")
	}
	data, info, err := evidencefile.ReadRegular(path, 1<<20)
	if err != nil || info.Mode().Perm()&0o077 != 0 {
		return PrivateConfig{}, errors.New("broker config must be a bounded owner-only regular file")
	}
	var c PrivateConfig
	if evidencefile.DecodeStrict(data, &c) != nil || c.Validate(a) != nil {
		return PrivateConfig{}, errors.New("private broker config does not match reviewed authority")
	}
	return c, nil
}

func (c PrivateConfig) Validate(a Authority) error {
	if a.Validate() != nil || c.Version != TransportVersion || c.RunID != a.RunID || c.PolicyDigest != a.PolicySHA256 || len(c.Capability) < 32 || !Identifier(c.Capability) || !filepath.IsAbs(c.SocketPath) || filepath.Clean(c.SocketPath) != c.SocketPath || len(c.SocketPath) > 100 || len(c.Invocations) != len(a.Operations) {
		return errors.New("invalid private broker identity")
	}
	for i, op := range a.Operations {
		inv := c.Invocations[i]
		if inv.StepID != op.StepID || inv.OperationID != op.OperationID || inv.InvocationID != op.InvocationID || !reflect.DeepEqual(inv.Bindings, op.Bindings) {
			return errors.New("private broker inventory mismatch")
		}
	}
	info, err := os.Lstat(c.SocketPath)
	if err != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0o077 != 0 {
		return errors.New("broker socket must be owner-only")
	}
	return nil
}
