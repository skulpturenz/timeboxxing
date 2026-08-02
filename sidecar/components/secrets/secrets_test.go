package secrets

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

var errKeyringUnavailable = errors.New("keyring unavailable")

// keyring.MockInit reassigns an unsynchronised process-global provider, and the mock it installs
// keeps its secrets in a map that is unsynchronised too — so distinct namespaces do not make these
// tests safe to interleave, because the race is on the one shared map rather than the values in it.
// Held from the test body to t.Cleanup, this keeps every test t.Parallel() while running the
// keyring section serially. Not reentrant: one useMockKeyring/useFailingKeyring per test.
var keyringMu sync.Mutex

// TestMain installs the mock before any test runs, so a test that forgets the fixture still cannot
// reach a real keychain — go-keyring offers no way back to the real provider once mocked.
func TestMain(m *testing.M) {
	keyring.MockInit()

	m.Run()
}

func useMockKeyring(t *testing.T) {
	t.Helper()

	keyringMu.Lock()
	t.Cleanup(keyringMu.Unlock)

	keyring.MockInit()
}

func useFailingKeyring(t *testing.T, err error) {
	t.Helper()

	keyringMu.Lock()
	t.Cleanup(keyringMu.Unlock)

	keyring.MockInitWithError(err)
}

func seedSecret(t *testing.T, namespace, key, value string) {
	t.Helper()

	require.NoError(t, keyring.Set(namespace, key, value), "seed secret %s/%s", namespace, key)
}

// storedSecret reads through keyring.Get rather than QueryGetSecret, so an assertion about what a
// write left behind does not depend on the query under test.
func storedSecret(t *testing.T, namespace, key string) (string, error) {
	t.Helper()

	return keyring.Get(namespace, key)
}
