package helpers

import "crypto/rand"

const otpPayloads = "0123456789"

func GenerateOTPCode(length int) (string, error) {
	otpCharsLength := byte(len(otpPayloads))
	// maxValid is the largest multiple of otpCharsLength that fits in a byte,
	// used to eliminate modulo bias when mapping random bytes to OTP digits.
	maxValid := 256 - (256 % int(otpCharsLength)) // 250 for 10 digits

	result := make([]byte, length)
	buf := make([]byte, length+10) // extra bytes for rejection sampling

	filled := 0
	for filled < length {
		_, err := rand.Read(buf)
		if err != nil {
			return "", err
		}
		for _, b := range buf {
			if filled >= length {
				break
			}
			if int(b) < maxValid {
				result[filled] = otpPayloads[b%otpCharsLength]
				filled++
			}
		}
	}

	return string(result), nil
}
