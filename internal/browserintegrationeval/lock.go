package browserintegrationeval

import (
	_ "embed"
	"fmt"

	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/browserverify"
)

//go:embed current-compatibility-lock-v2.json
var m86CurrentCompatibilityLock []byte

//go:embed current-compatibility-lock-v3.json
var currentV3CompatibilityLock []byte

//go:embed current-compatibility-lock-v4.json
var currentV4CompatibilityLock []byte

func decodeCompatibilityLock(data []byte) (browserscenario.CompatibilityLock, error) {
	var lock browserscenario.CompatibilityLock
	if err := browserverify.DecodeStrictJSON(data, &lock); err != nil {
		return lock, err
	}
	if err := browserscenario.ValidateCompatibilityLock(lock); err != nil {
		return lock, err
	}
	return lock, nil
}

func loadCurrentCompatibilityLock() (browserscenario.CompatibilityLock, error) {
	return decodeCompatibilityLock(currentV4CompatibilityLock)
}

func contractForVersion(version string) (browserscenario.CompatibilityLock, []gate, error) {
	switch version {
	case LegacyReportVersion:
		lock, err := browserscenario.LoadCompatibilityLock()
		return lock, legacyGates(), err
	case M86ReportVersion:
		lock, err := decodeCompatibilityLock(m86CurrentCompatibilityLock)
		return lock, currentGates(), err
	case CurrentV3ReportVersion:
		lock, err := decodeCompatibilityLock(currentV3CompatibilityLock)
		return lock, currentGates(), err
	case ReportVersion:
		lock, err := decodeCompatibilityLock(currentV4CompatibilityLock)
		return lock, currentGates(), err
	default:
		return browserscenario.CompatibilityLock{}, nil, fmt.Errorf("unsupported browser integration report version %q", version)
	}
}
