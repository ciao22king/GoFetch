package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type buildPlan struct {
	command string
	args    []string
	label   string
}

type buildResult struct {
	command  string
	output   string
	err      error
	duration time.Duration
	skipped  bool
}

// detectBuildPlan looks for a recognizable project manifest and returns the
// command that builds it. It intentionally avoids installing dependencies:
// only real build commands run, and only for repositories you chose to clone.
func detectBuildPlan(target string) buildPlan {
	switch {
	case exists(filepath.Join(target, "go.mod")):
		return buildPlan{command: "go", args: []string{"build", "./..."}, label: "go build ./..."}
	case exists(filepath.Join(target, "Cargo.toml")):
		return buildPlan{command: "cargo", args: []string{"build"}, label: "cargo build"}
	case exists(filepath.Join(target, "pom.xml")):
		return buildPlan{command: "mvn", args: []string{"-q", "-DskipTests", "package"}, label: "mvn package"}
	case exists(filepath.Join(target, "build.gradle")) || exists(filepath.Join(target, "build.gradle.kts")):
		return buildPlan{command: "gradle", args: []string{"build"}, label: "gradle build"}
	case exists(filepath.Join(target, "Makefile")) || exists(filepath.Join(target, "makefile")):
		return buildPlan{command: "make", label: "make"}
	case exists(filepath.Join(target, "pyproject.toml")):
		return buildPlan{command: "python3", args: []string{"-m", "build"}, label: "python -m build"}
	case exists(filepath.Join(target, "setup.py")):
		return buildPlan{command: "python3", args: []string{"setup.py", "build"}, label: "python setup.py build"}
	}
	return nodeBuildPlan(target)
}

type packageJSON struct {
	Scripts map[string]string `json:"scripts"`
}

func nodeBuildPlan(target string) buildPlan {
	manifest, ok := readPackageJSON(target)
	if !ok {
		return buildPlan{}
	}

	script := firstScript(manifest.Scripts, "build", "compile", "dist", "bundle")
	if script == "" {
		return buildPlan{}
	}

	switch detectNodeManager(target) {
	case "yarn":
		return buildPlan{command: "yarn", args: []string{script}, label: "yarn " + script}
	case "pnpm":
		return buildPlan{command: "pnpm", args: []string{"run", script}, label: "pnpm run " + script}
	default:
		return buildPlan{command: "npm", args: []string{"run", script}, label: "npm run " + script}
	}
}

func readPackageJSON(target string) (packageJSON, bool) {
	data, err := os.ReadFile(filepath.Join(target, "package.json"))
	if err != nil {
		return packageJSON{}, false
	}
	var manifest packageJSON
	if json.Unmarshal(data, &manifest) != nil {
		return packageJSON{}, false
	}
	return manifest, true
}

func firstScript(scripts map[string]string, names ...string) string {
	for _, name := range names {
		if strings.TrimSpace(scripts[name]) != "" {
			return name
		}
	}
	return ""
}

func detectNodeManager(target string) string {
	switch {
	case exists(filepath.Join(target, "pnpm-lock.yaml")):
		return "pnpm"
	case exists(filepath.Join(target, "yarn.lock")):
		return "yarn"
	default:
		return "npm"
	}
}

func buildRepository(target string) buildResult {
	plan := detectBuildPlan(target)
	if plan.command == "" {
		return buildResult{skipped: true}
	}

	started := time.Now()
	command := resolveCommand(plan.command)
	if command == "" {
		return buildResult{
			command:  plan.label,
			err:      fmt.Errorf("%s non trovato nel PATH", plan.command),
			duration: time.Since(started),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, plan.args...)
	cmd.Dir = target
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		err = fmt.Errorf("timeout dopo 10 minuti")
	} else if err != nil {
		if detail := lastMeaningfulLine(output.String()); detail != "" {
			err = fmt.Errorf("%v — %s", err, detail)
		}
	}

	return buildResult{
		command:  plan.label,
		output:   strings.TrimSpace(output.String()),
		err:      err,
		duration: time.Since(started),
	}
}

func lastMeaningfulLine(output string) string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for index := len(lines) - 1; index >= 0; index-- {
		if line := strings.TrimSpace(lines[index]); line != "" {
			return line
		}
	}
	return ""
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// resolveCommand returns the executable to run, or "" if none is available.
// Python is called python3 on macOS/Linux and python on Windows, so fall back
// to whichever exists.
func resolveCommand(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	if name == "python3" {
		if path, err := exec.LookPath("python"); err == nil {
			return path
		}
	}
	return ""
}
