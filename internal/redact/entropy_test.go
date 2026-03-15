package redact

import (
	"math"
	"testing"
)

func TestEntropy_EmptyString(t *testing.T) {
	if got := Entropy(""); got != 0 {
		t.Errorf("expected 0 for empty string, got %f", got)
	}
}

func TestEntropy_SingleChar(t *testing.T) {
	if got := Entropy("aaaa"); got != 0 {
		t.Errorf("expected 0 for single repeated char, got %f", got)
	}
}

func TestEntropy_UniformDistribution(t *testing.T) {
	// "ab" has 2 unique chars over 2 positions → entropy = 1.0 bits
	got := Entropy("ab")
	want := 1.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Entropy(\"ab\") = %f, want %f", got, want)
	}
}

func TestEntropy_NormalText(t *testing.T) {
	// Normal English words have low entropy
	cases := []string{"hello", "world", "password", "username", "delete"}
	for _, s := range cases {
		got := Entropy(s)
		if got >= 4.5 {
			t.Errorf("Entropy(%q) = %f, expected < 4.5 for normal word", s, got)
		}
	}
}

func TestEntropy_HighEntropySecrets(t *testing.T) {
	// Typical secrets have high entropy
	cases := []string{
		"ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ123456",       // GitHub PAT-like
		"AKIAIOSFODNN7EXAMPLE1234567890AB",             // AWS key-like
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",       // JWT header (base64)
		"sk-abcdefghijklmnopqrstuvwxyz1234567890ABCD", // OpenAI key-like
	}
	for _, s := range cases {
		got := Entropy(s)
		if got < 4.0 {
			t.Errorf("Entropy(%q) = %f, expected >= 4.0 for secret-like string", s, got)
		}
	}
}

func TestEntropyRedactor_RedactsHighEntropyToken(t *testing.T) {
	r := NewEntropyRedactor(4.5, 20)
	secret := "ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ123456"
	input := "curl -H Authorization:" + secret
	got := r.Redact(input)
	if got == input {
		t.Errorf("expected redaction, got unchanged: %q", got)
	}
	if contains(got, secret) {
		t.Errorf("secret still present after redaction: %q", got)
	}
}

func TestEntropyRedactor_PreservesNormalCommand(t *testing.T) {
	r := NewEntropyRedactor(4.5, 20)
	cases := []string{
		"ls -la /tmp",
		"git status",
		"echo hello world",
		"rm -rf /tmp/testdir",
		"docker ps --all",
	}
	for _, input := range cases {
		got := r.Redact(input)
		if got != input {
			t.Errorf("Redact(%q) = %q, expected unchanged", input, got)
		}
	}
}

func TestEntropyRedactor_MinLengthSkipsShortTokens(t *testing.T) {
	r := NewEntropyRedactor(3.0, 20) // low threshold but high min length
	// "abcdefgh" has high-ish entropy but is only 8 chars → should not be redacted
	input := "cmd abcdefghijkl"
	got := r.Redact(input)
	if got != input {
		t.Errorf("expected unchanged (too short), got %q", got)
	}
}

func TestEntropyRedactor_PreservesWhitespace(t *testing.T) {
	r := NewEntropyRedactor(4.5, 20)
	// Two spaces between tokens should be preserved
	input := "cmd  normal  word"
	got := r.Redact(input)
	if got != input {
		t.Errorf("whitespace not preserved: got %q, want %q", got, input)
	}
}

func TestEntropyRedactor_MultipleHighEntropyTokens(t *testing.T) {
	r := NewEntropyRedactor(4.5, 20)
	secret1 := "ghp_aBcDeFgHiJkLmNoPqRsTuVwXyZ123456"
	secret2 := "sk-abcdefghijklmnopqrstuvwxyz1234567890ABCD"
	input := secret1 + " " + secret2
	got := r.Redact(input)
	if contains(got, secret1) || contains(got, secret2) {
		t.Errorf("secrets still present after redaction: %q", got)
	}
}

func TestEntropyRedactor_ThresholdBoundary(t *testing.T) {
	// "ab" has exactly 1.0 bit entropy (2 unique chars, uniform distribution)
	// threshold uses >=, so at exactly threshold the token IS redacted
	r := NewEntropyRedactor(1.0, 2)
	got := r.Redact("ab")
	if got == "ab" {
		t.Errorf("token at exact threshold should be redacted (>= check), got unchanged")
	}

	// Token below threshold should not be redacted
	r2 := NewEntropyRedactor(1.1, 2)
	got2 := r2.Redact("ab")
	if got2 != "ab" {
		t.Errorf("token below threshold should not be redacted, got %q", got2)
	}
}

func TestEntropyRedactor_NilSafe(t *testing.T) {
	r := NewEntropyRedactor(4.5, 20)
	got := r.Redact("")
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func contains(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
