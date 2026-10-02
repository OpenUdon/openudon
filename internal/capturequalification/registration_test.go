package capturequalification

import (
	"context"
	"path/filepath"
	"testing"
)

func TestNeutralQualificationRefusesNonLoopbackAndMissingExecutableBeforeEffects(t *testing.T) {
	for _, target := range []string{"https://app.example.test/register", "http://192.0.2.1/register", "http://127.0.0.1/register#fragment", "http://private:secret@127.0.0.1/register"} {
		_, err := RunRegistration(context.Background(), RegistrationOptions{InitialURL: target, Origin: "http://127.0.0.1", Executable: "/missing", PrivateRoot: filepath.Join(t.TempDir(), "private")})
		if err == nil {
			t.Fatal("invalid test authority accepted", target)
		}
	}
}
