package app

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

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
		{
			name: "node compile script",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"compile":"tsc"}}`), 0o644)
			},
			expect: "npm run compile",
		},
		{
			name: "node with yarn lock",
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644)
			},
			expect: "yarn build",
		},
		{
			name: "node with pnpm lock",
			setup: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(`{"scripts":{"build":"vite build"}}`), 0o644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644)
			},
			expect: "pnpm run build",
		},
		{
			name: "maven",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pom.xml"), []byte("<project/>"), 0o644)
			},
			expect: "mvn package",
		},
		{
			name: "gradle",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "build.gradle"), []byte(""), 0o644)
			},
			expect: "gradle build",
		},
		{
			name: "make",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n"), 0o644)
			},
			expect: "make",
		},
		{
			name: "python pyproject",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[build-system]"), 0o644)
			},
			expect: "python -m build",
		},
		{
			name: "python setup.py",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "setup.py"), []byte(""), 0o644)
			},
			expect: "python setup.py build",
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

func TestDetectBuildPlanSkipsNodeWithoutBuildScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"scripts":{"test":"jest"}}`)
	if plan := detectBuildPlan(dir); plan.command != "" {
		t.Fatalf("detectBuildPlan() = %q, want no plan for a package.json without build script", plan.command)
	}
}

func TestBuildRepositorySkipsUnknownProject(t *testing.T) {
	result := buildRepository(t.TempDir())
	if !result.skipped {
		t.Fatalf("buildRepository() skipped = false, want true for unknown project")
	}
}
