package secrets

import (
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

func TestCommandDeleteSecret_RemovesTheStoredSecret(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.remove", "api-token", "s3cret")

	command := CommandDeleteSecret{Namespace: "com.timeboxxing.remove", Key: "api-token"}
	require.NoError(t, command.Exec(t.Context(), services.New()))

	_, err := storedSecret(t, "com.timeboxxing.remove", "api-token")
	assert.ErrorIs(t, err, keyring.ErrNotFound)
}

// Delete states a postcondition a key that was never set already satisfies. The original asserted on
// the ErrNotFound the keyring answers here, which panicked under -tags assert.
func TestCommandDeleteSecret_IsIdempotentForAMissingKey(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	command := CommandDeleteSecret{Namespace: "com.timeboxxing.idem", Key: "never-set"}
	require.NoError(t, command.Exec(t.Context(), services.New()))

	assert.NoError(t, command.Exec(t.Context(), services.New()), "deleting twice is not a failure either")
}

func TestCommandDeleteSecret_LeavesOtherKeysAlone(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.scope", "api-token", "s3cret")
	seedSecret(t, "com.timeboxxing.scope", "refresh-token", "keep-me")

	command := CommandDeleteSecret{Namespace: "com.timeboxxing.scope", Key: "api-token"}
	require.NoError(t, command.Exec(t.Context(), services.New()))

	stored, err := storedSecret(t, "com.timeboxxing.scope", "refresh-token")
	require.NoError(t, err)

	assert.Equal(t, "keep-me", stored)
}

func TestCommandDeleteSecret_SurfacesABackendFailure(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	command := CommandDeleteSecret{Namespace: "com.timeboxxing.fail", Key: "api-token"}
	err := command.Exec(t.Context(), services.New())

	assert.ErrorIs(t, err, errKeyringUnavailable)
}
