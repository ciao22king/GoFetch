package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ciao22king/GoFetch/internal/app"
)

var version = "0.6.0"

func main() {
	cfg := app.LoadConfig()

	var (
		repoURL = flag.String("url", "", "repository URL to clone")
		branch  = flag.String("branch", "", "branch to clone (defaults to the remote default branch)")
		dir     = flag.String("dir", cfg.DefaultDir, "parent directory for the clone")
		depth   = flag.Int("depth", cfg.Depth, "shallow clone depth (0 = full clone)")
		noBuild = flag.Bool("no-build", cfg.NoBuild, "skip the automatic build after cloning")
		yes     = flag.Bool("yes", false, "non-interactive mode: clone immediately, no TUI (requires -url)")
		force   = flag.Bool("force", false, "replace an existing destination in non-interactive mode")
		setDir  = flag.String("set-default-dir", "", "save a default clone directory to the config file and exit")
		showVer = flag.Bool("version", false, "print the version")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("gofetch %s\n", version)
		return
	}

	if *setDir != "" {
		cfg.DefaultDir = *setDir
		if err := app.SaveConfig(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "gofetch: non riesco a salvare la configurazione: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Directory predefinita salvata: %s\n", *setDir)
		return
	}

	initialDir := *dir
	if initialDir == "" {
		initialDir, _ = os.Getwd()
	}

	opts := app.Options{
		InitialURL:    *repoURL,
		InitialBranch: *branch,
		InitialDir:    initialDir,
		Version:       version,
		NoBuild:       *noBuild,
		Depth:         *depth,
		Overwrite:     *force,
	}

	var err error
	if *yes {
		if *repoURL == "" {
			fmt.Fprintln(os.Stderr, "gofetch: la modalità -yes richiede -url")
			os.Exit(2)
		}
		err = app.RunHeadless(opts)
	} else {
		err = app.Run(opts)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "gofetch: %v\n", err)
		os.Exit(1)
	}
}
