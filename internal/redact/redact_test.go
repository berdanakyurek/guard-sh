package redact

import "testing"

func TestRedactor_SinglePattern(t *testing.T) {
	r, err := New([]string{`(?i)password\s*=\s*\S+`})
	if err != nil {
		t.Fatal(err)
	}
	got := r.Redact("mysql -u root password=secret123")
	want := "mysql -u root [REDACTED]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedactor_NoMatch(t *testing.T) {
	r, err := New([]string{`password=\S+`})
	if err != nil {
		t.Fatal(err)
	}
	got := r.Redact("ls -la /tmp")
	if got != "ls -la /tmp" {
		t.Errorf("expected unchanged, got %q", got)
	}
}

func TestRedactor_MultiplePatterns(t *testing.T) {
	r, err := New([]string{`password=\S+`, `AKIA[0-9A-Z]{16}`})
	if err != nil {
		t.Fatal(err)
	}
	input := "export AWS_KEY=AKIA1234567890ABCDEF && mysql password=foo"
	got := r.Redact(input)
	if got == input {
		t.Error("expected redaction to occur")
	}
	want := "export AWS_KEY=[REDACTED] && mysql [REDACTED]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRedactor_InvalidPattern(t *testing.T) {
	_, err := New([]string{`[invalid`})
	if err == nil {
		t.Error("expected error for invalid pattern, got nil")
	}
}

func TestRedactor_EmptyPatterns(t *testing.T) {
	r, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	got := r.Redact("rm -rf /")
	if got != "rm -rf /" {
		t.Errorf("expected unchanged with no patterns, got %q", got)
	}
}

func TestRedactor_CaseInsensitive(t *testing.T) {
	r, err := New([]string{`(?i)password\s*=\s*\S+`})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		input string
		want  string
	}{
		{"set PASSWORD=abc", "set [REDACTED]"},
		{"set password=abc", "set [REDACTED]"},
		{"set Password=abc", "set [REDACTED]"},
	}
	for _, c := range cases {
		got := r.Redact(c.input)
		if got != c.want {
			t.Errorf("Redact(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestRedactor_MultipleMatchesSameLine(t *testing.T) {
	r, err := New([]string{`(?i)(password|secret)=\S+`})
	if err != nil {
		t.Fatal(err)
	}
	got := r.Redact("password=foo secret=bar")
	want := "[REDACTED] [REDACTED]"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
