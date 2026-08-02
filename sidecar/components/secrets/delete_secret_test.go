package secrets

import (
	"testing"

	"github.com/skulpturenz/timeboxxing/sidecar/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// The base case: after a delete the key reads back absent.
func TestCommandDeleteSecret_RemovesTheStoredSecret(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.remove", "api-token", "s3cret")

	err := CommandDeleteSecret{Namespace: "com.timeboxxing.remove", Key: "api-token"}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	_, err = storedSecret(t, "com.timeboxxing.remove", "api-token")
	assert.ErrorIs(t, err, keyring.ErrNotFound, "the secret is gone from the keyring itself")
}

// Delete states a postcondition — the key is not set — which a key that was never set already
// satisfies, so a retried or duplicated delete succeeds. The original asserted on the ErrNotFound
// the keyring answers here, which panicked under -tags assert.
func TestCommandDeleteSecret_IsIdempotentForAMissingKey(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)

	err := CommandDeleteSecret{Namespace: "com.timeboxxing.idem", Key: "never-set"}.Exec(t.Context(), services.New())
	require.NoError(t, err, "deleting an absent key is not a failure")

	err = CommandDeleteSecret{Namespace: "com.timeboxxing.idem", Key: "never-set"}.Exec(t.Context(), services.New())
	assert.NoError(t, err, "and neither is deleting it twice")
}

// Delete is scoped to one namespace/key pair, not to the namespace.
func TestCommandDeleteSecret_LeavesOtherKeysAlone(t *testing.T) {
	t.Parallel()

	useMockKeyring(t)
	seedSecret(t, "com.timeboxxing.scope", "api-token", "s3cret")
	seedSecret(t, "com.timeboxxing.scope", "refresh-token", "keep-me")

	err := CommandDeleteSecret{Namespace: "com.timeboxxing.scope", Key: "api-token"}.Exec(t.Context(), services.New())
	require.NoError(t, err)

	stored, err := storedSecret(t, "com.timeboxxing.scope", "refresh-token")
	require.NoError(t, err)

	assert.Equal(t, "keep-me", stored)
}

// A keyring that refuses the call is a real error, distinct from a key that is simply not there.
func TestCommandDeleteSecret_SurfacesABackendFailure(t *testing.T) {
	t.Parallel()

	useFailingKeyring(t, errKeyringUnavailable)

	err := CommandDeleteSecret{Namespace: "com.timeboxxing.fail", Key: "api-token"}.Exec(t.Context(), services.New())
	require.Error(t, err)

	assert.ErrorIs(t, err, errKeyringUnavailable)
}
