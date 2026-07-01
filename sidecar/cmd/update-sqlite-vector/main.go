package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/skulpturenz/timeboxxing/sidecar/internal/sqlitevectorupdate"
)

func main() {
	outputDir := flag.String("output", "db/sqlite-vector", "directory for bundled sqlite-vector extension files")
	releaseURL := flag.String("release-url", sqlitevectorupdate.DefaultReleaseURL, "GitHub release API URL")
	flag.Parse()

	updater := sqlitevectorupdate.Updater{ReleaseURL: *releaseURL}
	if err := updater.Update(context.Background(), *outputDir); err != nil {
		fmt.Fprintf(os.Stderr, "update sqlite-vector: %v\n", err)
		os.Exit(1)
	}
}
