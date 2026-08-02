package sqlitevector

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

//go:embed */vector.*
var sqliteVectorExtensionFiles embed.FS

// extractOnce memoizes the extraction for the life of the process. Load runs from the driver's
// ConnectHook, so it is called once per connection — without this, every connection would extract
// another copy of the extension.
//
//nolint:gochecknoglobals // process-wide memo for a process-wide side effect (see extract)
var extractOnce = sync.OnceValues(func() (*string, error) {
	resourcePath, ok := getFileName(runtime.GOOS, runtime.GOARCH)
	if !ok {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	// the extension is embedded in the binary
	// need to extract to a temp dir to load it
	return extract(*resourcePath)
})

const (
	sqliteVectorEntryPoint = "sqlite3_vector_init"
	// owner: read, write, execute
	// others: read, execute
	extensionFileMode = os.FileMode(0o755)
)

type Options struct {
	Path *string
}

func (options Options) Load() (*string, string, error) {
	if options.Path != nil && strings.TrimSpace(*options.Path) != "" {
		finalPath := *options.Path

		return &finalPath, sqliteVectorEntryPoint, nil
	}

	extractedPath, err := extractOnce()
	if err != nil {
		return nil, sqliteVectorEntryPoint, fmt.Errorf("unable to load sqlite-vector")
	}

	finalPath := *extractedPath

	return &finalPath, sqliteVectorEntryPoint, nil
}

func extract(resourcePath string) (*string, error) {
	data, err := sqliteVectorExtensionFiles.
		ReadFile(filepath.ToSlash(resourcePath))
	if err != nil {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	// A fresh directory per process, rather than one content-addressed path shared by every
	// process on the machine. Two processes extracting at once used to race on that shared path,
	// and a reader could load a half-written file — which is why the test suite had to run with
	// `go test -p 1`. MkdirTemp gives each process its own name, so there is nothing to race on.
	targetDir, err := os.MkdirTemp("", "timeboxxing-sqlite-vector-")
	if err != nil {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	targetPath := filepath.Join(targetDir, filepath.Base(resourcePath))
	if err := os.WriteFile(targetPath, data, extensionFileMode); err != nil {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	return &targetPath, nil
}

func getFileName(os string, arch string) (*string, bool) {
	var suffix string
	switch os {
	case "darwin":
		suffix = ".dylib"
	case "linux":
		suffix = ".so"
	case "windows":
		suffix = ".dll"
	default:
		return nil, false
	}

	switch arch {
	case "amd64", "arm64":
		filename := filepath.Join(fmt.Sprintf("%v-%v", os, arch), fmt.Sprintf("vector%v", suffix))

		return &filename, true
	default:
		return nil, false
	}
}
