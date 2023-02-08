package helpers

import "crypto/rand"

const otpPayloads = "0123456789"

func GenerateOTPCode(length int) (string, error) {
	buffer := make([]byte, length)
	_, err := rand.Read(buffer)
	if err != nil {
		return "", err
	}

	otpCharsLength := len(otpPayloads)
	for i := 0; i < length; i++ {
		buffer[i] = otpPayloads[int(buffer[i])%otpCharsLength]
	}

	return string(buffer), nil
}
