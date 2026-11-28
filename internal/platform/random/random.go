// Package random draws the short codes and the tokens a service hands out, from crypto/rand.
package random

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"math"
)

const (
	codeSpace  = 1_000_000
	tokenBytes = 32
)

// codeBound is the largest multiple of codeSpace a uint32 holds. A draw at or above it is drawn again, so every
// code is equally likely.
const codeBound = math.MaxUint32 / codeSpace * codeSpace

// Generator draws codes and tokens from crypto/rand. The zero value is ready to use, and it is safe for
// concurrent use.
type Generator struct{}

// Code returns six decimal digits, leading zeros kept, each of the million codes equally likely.
func (Generator) Code() string {
	var draw [4]byte
	for {
		_, _ = rand.Read(draw[:])
		if value := binary.BigEndian.Uint32(draw[:]); value < codeBound {
			return fmt.Sprintf("%06d", value%codeSpace)
		}
	}
}

// Token returns 32 random bytes in unpadded base64url: 43 characters that are safe in a URL.
func (Generator) Token() string {
	var raw [tokenBytes]byte
	_, _ = rand.Read(raw[:])
	return base64.RawURLEncoding.EncodeToString(raw[:])
}
