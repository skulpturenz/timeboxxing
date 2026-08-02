package secrets

import (
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// errKeyringUnavailable stands in for whatever the OS keyring daemon returns when it is locked,
// missing or refuses the call. The component has to surface it, never assert it away.
var errKeyringUnavailable = errors.New("keyring unavailable")

// keyringMu serialises this package's tests against go-keyring's process-global provider.
//
// keyring.MockInit reassigns an unsynchronised package-level provider, and the mock it installs
// keeps its secrets in a map that is unsynchronised too — so two tests in flight at once race on
// both, and distinct namespaces do not help because the race is on the one shared map, not on the
// values in it. Taking this lock in the test body and releasing it from t.Cleanup holds it for the
// whole test, which keeps every test t.Parallel() as the house style requires while making the
// keyring section serial in practice. Not reentrant: one useMockKeyring/useFailingKeyring per test.
var keyringMu sync.Mutex

// TestMain installs the mock before any test runs so that a test which forgets useMockKeyring
// cannot reach a real keychain. go-keyring offers no way back to the real provider once mocked,
// which is what makes this a safe floor: CI's linux runner is headless with no Secret Service, and
// on a developer's machine the real calls would prompt against — or write into — their login
// keychain.
func TestMain(m *testing.M) {
	keyring.MockInit()

	m.Run()
}

// useMockKeyring swaps in go-keyring's in-memory provider, starting empty.
func useMockKeyring(t *testing.T) {
	t.Helper()

	keyringMu.Lock()
	t.Cleanup(keyringMu.Unlock)

	keyring.MockInit()
}

// useFailingKeyring is useMockKeyring with a provider that fails every call, for the paths where
// the backend itself is the error.
func useFailingKeyring(t *testing.T, err error) {
	t.Helper()

	keyringMu.Lock()
	t.Cleanup(keyringMu.Unlock)

	keyring.MockInitWithError(err)
}

// seedSecret writes through the same keyring.Set the component uses, so a fixture can never
// diverge from the production write path.
func seedSecret(t *testing.T, namespace, key, value string) {
	t.Helper()

	require.NoError(t, keyring.Set(namespace, key, value), "seed secret %s/%s", namespace, key)
}

// storedSecret reads through keyring.Get directly rather than through QueryGetSecret, so an
// assertion about what a write left behind does not depend on the query under test.
func storedSecret(t *testing.T, namespace, key string) (string, error) {
	t.Helper()

	return keyring.Get(namespace, key)
}
