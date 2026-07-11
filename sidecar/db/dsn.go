package db

import (
	"fmt"
	"net/url"
	"time"

	enumsjournalmode "github.com/skulpturenz/timeboxxing/sidecar/enums/enums_journal_mode"
	"github.com/skulpturenz/timeboxxing/sidecar/envs"
)

type DSN struct {
	path              string
	journalMode       enumsjournalmode.JournalMode
	enableForeignKeys bool
	enableEncryption  bool
	encryptionKey     *string
	busyTimeout       *time.Time
}

func NewDSN(path string) DSN {
	pathWithEnv := fmt.Sprintf("%v_%v", path, envs.GO_ENV.Value())
	return DSN{path: pathWithEnv}
}

func (dsn *DSN) SetJournalMode(mode enumsjournalmode.JournalMode) {
	dsn.journalMode = mode
}

func (dsn *DSN) SetBusyTimeout(timeout time.Time) {
	busyTimeout := timeout
	dsn.busyTimeout = &busyTimeout
}

func (dsn *DSN) EnableFK() {
	dsn.enableForeignKeys = true
}

func (dsn *DSN) EnableEncryption(key string) error {
	if !isHexKey(key) {
		return fmt.Errorf("encryption key is not hex")
	}

	dsn.enableEncryption = true

	encryptionKey := key
	dsn.encryptionKey = &encryptionKey

	return nil
}

func (dsn *DSN) IsEncrypted() bool {
	return dsn.enableEncryption == true && dsn.encryptionKey != nil
}

func (dsn *DSN) String() string {
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

	return fmt.Sprintf("%v?%v", dsn.path, params.Encode())
}

func isHexKey(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9', r >= 'a' && r <= 'f', r >= 'A' && r <= 'F':
		default:
			return false
		}
	}
	return true
}
