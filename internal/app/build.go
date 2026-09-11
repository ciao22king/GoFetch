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

func detectBuildPlan(target string) buildPlan {
	if exists(filepath.Join(target, "go.mod")) {
		return buildPlan{command: "go", args: []string{"build", "./..."}, label: "go build ./..."}
	}
	if exists(filepath.Join(target, "Cargo.toml")) {
		return buildPlan{command: "cargo", args: []string{"build"}, label: "cargo build"}
	}
	if exists(filepath.Join(target, "package.json")) {
		var packageJSON struct {
			Scripts map[string]string `json:"scripts"`
		}
		data, err := os.ReadFile(filepath.Join(target, "package.json"))
		if err == nil && json.Unmarshal(data, &packageJSON) == nil {
			if strings.TrimSpace(packageJSON.Scripts["build"]) != "" {
				return buildPlan{command: "npm", args: []string{"run", "build"}, label: "npm run build"}
			}
		}
	}
	return buildPlan{}
}

func buildRepository(target string) buildResult {
	plan := detectBuildPlan(target)
	if plan.command == "" {
		return buildResult{skipped: true}
	}

	started := time.Now()
	if _, err := exec.LookPath(plan.command); err != nil {
		return buildResult{
			command:  plan.label,
			err:      fmt.Errorf("%s non trovato nel PATH", plan.command),
			duration: time.Since(started),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, plan.command, plan.args...)
	cmd.Dir = target
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	if ctx.Err() != nil {
		err = fmt.Errorf("timeout dopo 10 minuti")
	}

	return buildResult{
		command:  plan.label,
		output:   strings.TrimSpace(output.String()),
		err:      err,
		duration: time.Since(started),
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
