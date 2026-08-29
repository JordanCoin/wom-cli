package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveSecretPrefersTheFlag(t *testing.T) {
	t.Setenv(envVerificationCode, "from-env")

	if got := resolveSecret("from-flag", envVerificationCode); got != "from-flag" {
		t.Fatalf("flag should win over the environment, got %q", got)
	}
}

func TestResolveSecretFallsBackToTheEnvironment(t *testing.T) {
	t.Setenv(envVerificationCode, "from-env")

	if got := resolveSecret("", envVerificationCode); got != "from-env" {
		t.Fatalf("empty flag should fall back to the environment, got %q", got)
	}
}

func TestResolveSecretIsEmptyWhenNeitherIsSet(t *testing.T) {
	t.Setenv(envVerificationCode, "")

	if got := resolveSecret("", envVerificationCode); got != "" {
		t.Fatalf("nothing set should resolve to empty, got %q", got)
	}
}

func TestResolveSecretReadsTheGroupIDFromTheEnvironmentToo(t *testing.T) {
	t.Setenv(envGroupID, "5165")

	if got := resolveSecret("", envGroupID); got != "5165" {
		t.Fatalf("group id should fall back to the environment, got %q", got)
	}
}

// A group competition's create response echoes back the code the caller
// already sent. Printing it would copy a long-lived group secret into
// stdout, and from there into a transcript or an agent's tool log.
func TestCreateOutputNeverEchoesASuppliedCode(t *testing.T) {
	const secret = "123-456-789"

	for _, asJSON := range []bool{false, true} {
		data := map[string]interface{}{
			"competition":      map[string]interface{}{"id": float64(88345)},
			"verificationCode": secret,
		}
		var out bytes.Buffer
		if err := renderCreateOutput(&out, data, "Summer Bingo", "ehb", "s", "e", secret, asJSON); err != nil {
			t.Fatalf("render failed: %s", err)
		}
		if strings.Contains(out.String(), secret) {
			t.Fatalf("json=%v printed the supplied verification code:\n%s", asJSON, out.String())
		}
		if !strings.Contains(out.String(), "88345") {
			t.Fatalf("json=%v dropped the competition id:\n%s", asJSON, out.String())
		}
	}
}

// The other half of the same rule: a standalone competition mints its own
// code, and the create response is the only place anyone will ever see it.
func TestCreateOutputPrintsAFreshStandaloneCode(t *testing.T) {
	const minted = "999-888-777"

	for _, asJSON := range []bool{false, true} {
		data := map[string]interface{}{
			"competition":      map[string]interface{}{"id": float64(12)},
			"verificationCode": minted,
		}
		var out bytes.Buffer
		if err := renderCreateOutput(&out, data, "RC SOTW", "runecrafting", "s", "e", "", asJSON); err != nil {
			t.Fatalf("render failed: %s", err)
		}
		if !strings.Contains(out.String(), minted) {
			t.Fatalf("json=%v swallowed the only copy of a fresh code:\n%s", asJSON, out.String())
		}
	}
}
