package secrets

import (
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The round trip is the whole contract: what the keyring was given is what the query hands back.
func TestQueryGetSecret_ReadsBackTheStoredValue(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.read", "api-token", "s3cret")

	value, err := QueryGetSecret{Namespace: "com.timeboxxing.read", Key: "api-token"}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	require.NotNil(t, value, "a stored secret reads back as present")
	assert.Equal(t, "s3cret", *value)
}

// A keyring miss is external state, not a failure — it reads back as nil with no error. The
// original asserted the ErrNotFound away before tolerating it, so this ordinary outcome panicked
// under -tags assert, which is exactly how CI builds.
func TestQueryGetSecret_ReportsAMissAsAbsentNotAnError(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	value, err := QueryGetSecret{Namespace: "com.timeboxxing.miss", Key: "never-set"}.Exec(t.Context(), services.New())
	require.NoError(t, err, "a key that was never set is absent, not an error")

	assert.Nil(t, value)
}

// Documented consequence of utils.ZeroNil: a secret deliberately stored as the empty string is
// indistinguishable from one that was never set, so callers must not use one to mean "set but
// blank".
func TestQueryGetSecret_ReadsAnEmptyValueAsAbsent(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.empty", "blank", "")

	value, err := QueryGetSecret{Namespace: "com.timeboxxing.empty", Key: "blank"}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	assert.Nil(t, value, "an empty stored value collapses to absent")
}

// A locked or missing keyring daemon is a real error path, wrapped so [errors.Is] still finds the
// cause.
func TestQueryGetSecret_SurfacesABackendFailure(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	value, err := QueryGetSecret{Namespace: "com.timeboxxing.fail", Key: "api-token"}.Exec(t.Context(), services.New())
	require.Error(t, err)

	require.ErrorIs(t, err, errKeyringUnavailable, "the backend failure reaches the caller intact")
	assert.Nil(t, value, "a failed read reports no value at all")
}

// Namespace is half the identity: the same key under two namespaces is two secrets, not one.
func TestQueryGetSecret_KeepsNamespacesApart(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.one", "api-token", "first")
	seedSecret(t, "com.timeboxxing.two", "api-token", "second")

	first, err := QueryGetSecret{Namespace: "com.timeboxxing.one", Key: "api-token"}.Exec(t.Context(), services.New())
	require.NoError(t, err)
	require.NotNil(t, first)

	second, err := QueryGetSecret{Namespace: "com.timeboxxing.two", Key: "api-token"}.Exec(t.Context(), services.New())
	require.NoError(t, err)
	require.NotNil(t, second)

	assert.Equal(t, "first", *first)
	assert.Equal(t, "second", *second)
}
