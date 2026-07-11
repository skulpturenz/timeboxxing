package sqlitevector

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed */vector.*
var sqliteVectorExtensionFiles embed.FS

const (
	sqliteVectorEntryPoint = "sqlite3_vector_init"
)

type Options struct {
	Path *string
}

func (options Options) Load() (loadedPath *string, entrypoint string, error error) {
	finalPath := ""
	if options.Path != nil && strings.TrimSpace(*options.Path) != "" {
		finalPath = *options.Path
	} else {
		resourcePath, ok := getFileName(runtime.GOOS, runtime.GOARCH)
		if !ok {
			return nil, sqliteVectorEntryPoint, fmt.Errorf("unable to load sqlite-vector")
		}

		// the extension is embedded in the binary
		// need to extract to a temp dir to load it
		extractedPath, err := extract(*resourcePath)
		if err != nil {
			return nil, sqliteVectorEntryPoint, fmt.Errorf("unable to load sqlite-vector")
		}

		finalPath = *extractedPath
	}

	return &finalPath, sqliteVectorEntryPoint, nil
}

func extract(resourcePath string) (*string, error) {
	data, err := sqliteVectorExtensionFiles.
		ReadFile(filepath.ToSlash(filepath.Join("sqlite-vector", resourcePath)))
	if err != nil {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	sum := sha256.Sum256(data)
	// owner: read, write, execute
	// others: read, execute
	fileMode := os.FileMode(0o755)

	targetDir := filepath.Join(
		os.TempDir(),
		"timeboxxing-sqlite-vector",
		hex.EncodeToString(sum[:8]),
		filepath.Dir(resourcePath),
	)
	if err := os.MkdirAll(targetDir, fileMode); err != nil {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	targetPath := filepath.Join(targetDir, filepath.Base(resourcePath))
	if existing, err := os.ReadFile(targetPath); err == nil && bytes.Equal(existing, data) {
		return &targetPath, fmt.Errorf("unable to load sqlite-vector")
	}
	if err := os.WriteFile(targetPath, data, fileMode); err != nil {
		return nil, fmt.Errorf("unable to load sqlite-vector")
	}

	return &targetPath, nil
}

func getFileName(os string, arch string) (*string, bool) {
	suffix := ""
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
