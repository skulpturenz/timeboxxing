package secrets

import (
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryGetSecret_ReadsBackTheStoredValue(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.read", "api-token", "s3cret")

	query := QueryGetSecret{Namespace: "com.timeboxxing.read", Key: "api-token"}
	value, err := query.Exec(t.Context(), services.New())
	require.NoError(t, err)
	require.NotNil(t, value)

	assert.Equal(t, "s3cret", *value)
}

// The original asserted the ErrNotFound away before tolerating it, so this ordinary outcome panicked
// under -tags assert, which is how CI builds.
func TestQueryGetSecret_ReportsAMissAsAbsentNotAnError(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	query := QueryGetSecret{Namespace: "com.timeboxxing.miss", Key: "never-set"}
	value, err := query.Exec(t.Context(), services.New())
	require.NoError(t, err)

	assert.Nil(t, value)
}

func TestQueryGetSecret_ReadsAnEmptyValueAsAbsent(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.empty", "blank", "")

	query := QueryGetSecret{Namespace: "com.timeboxxing.empty", Key: "blank"}
	value, err := query.Exec(t.Context(), services.New())
	require.NoError(t, err)

	assert.Nil(t, value, "utils.ZeroNil collapses a stored empty string to absent")
}

func TestQueryGetSecret_SurfacesABackendFailure(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	query := QueryGetSecret{Namespace: "com.timeboxxing.fail", Key: "api-token"}
	value, err := query.Exec(t.Context(), services.New())
	require.ErrorIs(t, err, errKeyringUnavailable)

	assert.Nil(t, value)
}

func TestQueryGetSecret_KeepsNamespacesApart(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.one", "api-token", "first")
	seedSecret(t, "com.timeboxxing.two", "api-token", "second")

	one := QueryGetSecret{Namespace: "com.timeboxxing.one", Key: "api-token"}
	first, err := one.Exec(t.Context(), services.New())
	require.NoError(t, err)
	require.NotNil(t, first)

	two := QueryGetSecret{Namespace: "com.timeboxxing.two", Key: "api-token"}
	second, err := two.Exec(t.Context(), services.New())
	require.NoError(t, err)
	require.NotNil(t, second)

	assert.Equal(t, "first", *first)
	assert.Equal(t, "second", *second)
}
