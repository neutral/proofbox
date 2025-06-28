package types

import "errors"

var (
	// ErrInvalidHashSize indicates a hash byte slice is not the correct size
	ErrInvalidHashSize = errors.New("invalid hash size: must be 32 bytes")
)
