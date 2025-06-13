package gohash

import (
	"encoding/hex"
)

// Short is the first four bytes of [Hash].
type Short [4]byte

// String returns a hex representation of the [Hash].
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Short returns short hash from [Hash].
func (h Hash) Short() Short {
	return Short(h[:4])
}

// String returns a hex representation of the [Short].
func (h Short) String() string {
	return hex.EncodeToString(h[:])
}
