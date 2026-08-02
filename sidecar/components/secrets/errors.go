package secrets

import "errors"

// ErrSecretExists is returned when a write is refused because the key is already set.
var ErrSecretExists = errors.New("secret already exists")
