package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectBuildPlan(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(string) error
		expect string
	}{
		{
			name: "go",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/test\n"), 0o644)
			},
			expect: "go build ./...",
		},
		{
			name: "rust",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte("[package]\nname = \"test\"\n"), 0o644)
			},
			expect: "cargo build",
		},
		{
			name: "node build script",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644)
			},
			expect: "npm run build",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.setup(dir); err != nil {
				t.Fatal(err)
			}
			plan := detectBuildPlan(dir)
			if plan.label != tt.expect {
				t.Fatalf("detectBuildPlan() = %q, want %q", plan.label, tt.expect)
			}
		})
	}
}

func TestDetectBuildPlanSkipsUnknownProject(t *testing.T) {
	plan := detectBuildPlan(t.TempDir())
	if plan.command != "" {
		t.Fatalf("detectBuildPlan() command = %q, want empty", plan.command)
	}
}
