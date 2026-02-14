package formatter

import (
	"testing"
)

func TestSubstituteArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		filePath string
		want     []string
	}{
		{
			name:     "no placeholders",
			args:     []string{"--write"},
			filePath: "/tmp/test.go",
			want:     []string{"--write"},
		},
		{
			name:     "single placeholder",
			args:     []string{"--stdin-filepath", "{file}"},
			filePath: "/tmp/test.go",
			want:     []string{"--stdin-filepath", "/tmp/test.go"},
		},
		{
			name:     "multiple placeholders",
			args:     []string{"{file}", "--output", "{file}.bak"},
			filePath: "/tmp/test.go",
			want:     []string{"/tmp/test.go", "--output", "/tmp/test.go.bak"},
		},
		{
			name:     "empty args",
			args:     []string{},
			filePath: "/tmp/test.go",
			want:     []string{},
		},
		{
			name:     "nil args",
			args:     nil,
			filePath: "/tmp/test.go",
			want:     []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SubstituteArgs(tt.args, tt.filePath)
			if len(got) != len(tt.want) {
				t.Fatalf("SubstituteArgs() returned %d args, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("SubstituteArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestSubstituteFilepathArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		filePath string
		want     []string
	}{
		{
			name:     "with placeholder",
			args:     []string{"--input", "{file}"},
			filePath: "/tmp/test.go",
			want:     []string{"--input", "/tmp/test.go"},
		},
		{
			name:     "without placeholder appends path",
			args:     []string{"--write"},
			filePath: "/tmp/test.go",
			want:     []string{"--write", "/tmp/test.go"},
		},
		{
			name:     "empty args appends path",
			args:     []string{},
			filePath: "/tmp/test.go",
			want:     []string{"/tmp/test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := substituteFilepathArgs(tt.args, tt.filePath)
			if len(got) != len(tt.want) {
				t.Fatalf("substituteFilepathArgs() returned %d args, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("substituteFilepathArgs()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
