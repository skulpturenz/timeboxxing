package db

import (
	"fmt"
	"net/url"
	"time"

	enumsjournalmode "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_journal_mode"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
)

type dsn struct {
	path              string
	journalMode       enumsjournalmode.JournalMode
	enableForeignKeys bool
	enableEncryption  bool
	encryptionKey     *string
	busyTimeout       *time.Time
}

func NewDSN(path string) dsn {
	pathWithEnv := fmt.Sprintf("%v_%v", path, envs.GO_ENV.Value())
	return dsn{path: pathWithEnv}
}

func (dsn *dsn) SetJournalMode(mode enumsjournalmode.JournalMode) {
	dsn.journalMode = mode
}

func (dsn *dsn) SetBusyTimeout(timeout time.Time) {
	busyTimeout := timeout
	dsn.busyTimeout = &busyTimeout
}

func (dsn *dsn) EnableFK() {
	dsn.enableForeignKeys = true
}

func (dsn *dsn) EnableEncryption(key string) {
	dsn.enableEncryption = true

	encryptionKey := key
	dsn.encryptionKey = &encryptionKey
}

func (dsn *dsn) String() {
	params := url.Values{}

	params.Set("_journal_mode", fmt.Sprintf("%v", dsn.journalMode))

	if dsn.enableForeignKeys {
		params.Set("_foreign_keys", "on")
	}

	if dsn.enableEncryption {
		params.Set("_cipher", "sqlcipher")
		params.Set("_key", fmt.Sprintf("x'%v'", dsn.encryptionKey))
	}

	if dsn.busyTimeout != nil {
		params.Set("_busy_timeout", fmt.Sprintf("%v", dsn.busyTimeout.UnixMilli()))
	}
}
