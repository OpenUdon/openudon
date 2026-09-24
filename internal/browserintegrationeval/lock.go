package browserintegrationeval

import (
	_ "embed"
	"fmt"

	"github.com/OpenUdon/openudon/internal/browserscenario"
	"github.com/OpenUdon/openudon/internal/browserverify"
)

//go:embed current-compatibility-lock-v2.json
var m86CurrentCompatibilityLock []byte

//go:embed current-compatibility-lock.json
var currentCompatibilityLock []byte

func loadCurrentCompatibilityLock() (browserscenario.CompatibilityLock, error) {
	var lock browserscenario.CompatibilityLock
	if err := browserverify.DecodeStrictJSON(currentCompatibilityLock, &lock); err != nil {
		return lock, err
	}
	if err := browserscenario.ValidateCompatibilityLock(lock); err != nil {
		return lock, err
	}
	return lock, nil
}

func loadM86CurrentCompatibilityLock() (browserscenario.CompatibilityLock, error) {
	var lock browserscenario.CompatibilityLock
	if err := browserverify.DecodeStrictJSON(m86CurrentCompatibilityLock, &lock); err != nil {
		return lock, err
	}
	if err := browserscenario.ValidateCompatibilityLock(lock); err != nil {
		return lock, err
	}
	return lock, nil
}

func contractForVersion(version string) (browserscenario.CompatibilityLock, []gate, error) {
	switch version {
	case LegacyReportVersion:
		lock, err := browserscenario.LoadCompatibilityLock()
		return lock, legacyGates(), err
	case ReportVersion:
		lock, err := loadM86CurrentCompatibilityLock()
		return lock, currentGates(), err
	default:
		return browserscenario.CompatibilityLock{}, nil, fmt.Errorf("unsupported browser integration report version %q", version)
	}
}
