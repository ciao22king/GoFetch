package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ciao22king/GoFetch/internal/app"
)

var version = "0.4.0"

func main() {
	var (
		repoURL = flag.String("url", "", "repository URL to clone")
		dir     = flag.String("dir", "", "parent directory for the clone")
		noBuild = flag.Bool("no-build", false, "skip the automatic build after cloning")
		showVer = flag.Bool("version", false, "print the version")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("gofetch %s\n", version)
		return
	}

	initialDir := *dir
	if initialDir == "" {
		initialDir, _ = os.Getwd()
	}

	if err := app.Run(app.Options{
		InitialURL: *repoURL,
		InitialDir: initialDir,
		Version:    version,
		NoBuild:    *noBuild,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "gofetch: %v\n", err)
		os.Exit(1)
	}
}
