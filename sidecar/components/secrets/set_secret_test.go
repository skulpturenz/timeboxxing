package secrets

import (
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The base case, and the only path the original guard let through — by accident.
func TestCommandSetSecret_WritesANewSecret(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.write",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: false,
	}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	stored, err := storedSecret(t, "com.timeboxxing.write", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "s3cret", stored)
}

// Overwrite is the caller's explicit permission to replace a live credential; without it the write
// is refused. The original guard was inverted, so Overwrite=false skipped the existence check
// entirely and clobbered silently.
func TestCommandSetSecret_RefusesToClobberWithoutOverwrite(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.clobber", "api-token", "original")

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.clobber",
		Key:       "api-token",
		Value:     "replacement",
		Overwrite: false,
	}.Exec(t.Context(), services.New())
	require.Error(t, err)

	assert.ErrorIs(t, err, ErrSecretExists, "the caller branches on this to confirm, then retries with Overwrite")
}

// Refusing has to be a no-op rather than a partial write: the existence probe must leave the stored
// value exactly as it found it.
func TestCommandSetSecret_LeavesTheStoredValueIntactWhenItRefuses(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.intact", "api-token", "original")

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.intact",
		Key:       "api-token",
		Value:     "replacement",
		Overwrite: false,
	}.Exec(t.Context(), services.New())
	require.Error(t, err)

	stored, err := storedSecret(t, "com.timeboxxing.intact", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "original", stored, "a refused write does not touch the keyring")
}

// The one path allowed to replace a secret. The original always errored here, so nothing could
// ever be rotated.
func TestCommandSetSecret_ReplacesWithOverwrite(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.replace", "api-token", "original")

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.replace",
		Key:       "api-token",
		Value:     "replacement",
		Overwrite: true,
	}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	stored, err := storedSecret(t, "com.timeboxxing.replace", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "replacement", stored)
}

// Overwrite is permission to replace, not a requirement that something already be there.
func TestCommandSetSecret_WritesWithOverwriteWhenTheKeyIsAbsent(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.absent",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: true,
	}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	stored, err := storedSecret(t, "com.timeboxxing.absent", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "s3cret", stored)
}

// A keyring that cannot answer the existence probe is not an absent key. Reading it as one would
// clobber a secret we merely failed to read, so the write is abandoned instead.
func TestCommandSetSecret_SurfacesAFailureReadingTheExistingValue(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.probefail",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: false,
	}.Exec(t.Context(), services.New())
	require.Error(t, err)

	require.ErrorIs(t, err, errKeyringUnavailable)
	assert.NotErrorIs(t, err, ErrSecretExists, "an unreadable keyring is not the same as an occupied key")
}

// With Overwrite the probe is skipped, so this is the write itself failing.
func TestCommandSetSecret_SurfacesABackendFailure(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	err := CommandSetSecret{
		Namespace: "com.timeboxxing.writefail",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: true,
	}.Exec(t.Context(), services.New())
	require.Error(t, err)

	assert.ErrorIs(t, err, errKeyringUnavailable)
}
