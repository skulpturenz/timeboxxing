package secrets

import (
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCommandSetSecret_WritesANewSecret(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.write",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: false,
	}
	require.NoError(t, command.Exec(t.Context(), services.New()))

	stored, err := storedSecret(t, "com.timeboxxing.write", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "s3cret", stored)
}

// The original guard was inverted, so Overwrite=false skipped the existence check entirely and
// replaced a live credential silently.
func TestCommandSetSecret_RefusesToClobberWithoutOverwrite(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.clobber", "api-token", "original")

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.clobber",
		Key:       "api-token",
		Value:     "replacement",
		Overwrite: false,
	}
	err := command.Exec(t.Context(), services.New())

	assert.ErrorIs(t, err, ErrSecretExists)
}

func TestCommandSetSecret_LeavesTheStoredValueIntactWhenItRefuses(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.intact", "api-token", "original")

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.intact",
		Key:       "api-token",
		Value:     "replacement",
		Overwrite: false,
	}
	require.Error(t, command.Exec(t.Context(), services.New()))

	stored, err := storedSecret(t, "com.timeboxxing.intact", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "original", stored, "the existence probe has no side effects")
}

// The original always errored here, so the one path allowed to replace a secret never could.
func TestCommandSetSecret_ReplacesWithOverwrite(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.replace", "api-token", "original")

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.replace",
		Key:       "api-token",
		Value:     "replacement",
		Overwrite: true,
	}
	require.NoError(t, command.Exec(t.Context(), services.New()))

	stored, err := storedSecret(t, "com.timeboxxing.replace", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "replacement", stored)
}

func TestCommandSetSecret_WritesWithOverwriteWhenTheKeyIsAbsent(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.absent",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: true,
	}
	require.NoError(t, command.Exec(t.Context(), services.New()))

	stored, err := storedSecret(t, "com.timeboxxing.absent", "api-token")
	require.NoError(t, err)

	assert.Equal(t, "s3cret", stored)
}

// A keyring that cannot answer the probe is not an absent key: reading it as one would clobber a
// secret we merely failed to read.
func TestCommandSetSecret_SurfacesAFailureReadingTheExistingValue(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.probefail",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: false,
	}
	err := command.Exec(t.Context(), services.New())
	require.ErrorIs(t, err, errKeyringUnavailable)

	assert.NotErrorIs(t, err, ErrSecretExists)
}

func TestCommandSetSecret_SurfacesABackendFailure(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	command := CommandSetSecret{
		Namespace: "com.timeboxxing.writefail",
		Key:       "api-token",
		Value:     "s3cret",
		Overwrite: true,
	}
	err := command.Exec(t.Context(), services.New())

	assert.ErrorIs(t, err, errKeyringUnavailable)
}
