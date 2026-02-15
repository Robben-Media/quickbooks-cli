package config

import (
	"testing"
)

func TestNormalizeEnvVarName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"quickbooks-cli", "QUICKBOOKS_CLI"},
		{"my-app", "MY_APP"},
		{"simple", "SIMPLE"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeEnvVarName(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeEnvVarName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAppName(t *testing.T) {
	if AppName != "quickbooks-cli" {
		t.Errorf("AppName = %q, want 'quickbooks-cli'", AppName)
	}
}

func TestConfigDir(t *testing.T) {
	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error: %v", err)
	}

	if dir == "" {
		t.Error("ConfigDir() returned empty string")
	}
}
