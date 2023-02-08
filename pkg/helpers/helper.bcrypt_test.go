package helpers_test

import (
	"fmt"
	"testing"

	"github.com/snykk/go-rest-boilerplate/pkg/helpers"
)

func TestGenerateHash(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedErr bool
	}{
		{"Success", "secret", false},
		{"Error", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			hash, err := helpers.GenerateHash(test.input)
			if (err != nil) != test.expectedErr {
				t.Errorf("GenerateHash(%q) error = %v, expectedErr %v", test.input, err, test.expectedErr)
			}
			if !test.expectedErr {
				if len(hash) == 0 {
					t.Errorf("GenerateHash(%q) = %q, expected non-empty string", test.input, hash)
				}
			}
		})
	}
}

func TestValidateHash(t *testing.T) {
	secret := "secret"
	poorHash := "mwehehehe"
	hash, err := helpers.GenerateHash(secret)
	if err != nil {
		t.Error(err)
	}
	tests := []struct {
		name    string
		secret  string
		hash    string
		isValid bool
	}{
		{"Success", secret, hash, true},
		{"Invalid", secret, hash[:len(hash)-len(poorHash)] + poorHash, false},
		{"Invalid", "invalid", "awkakwkawk", false},
	}
	for index, test := range tests {
		t.Run(fmt.Sprintf("Test %d | %s", index, test.name), func(t *testing.T) {
			if got := helpers.ValidateHash(test.secret, test.hash); got != test.isValid {
				t.Errorf("ValidateHash(%q, %q) = %v, expected %v", test.secret, test.hash, got, test.isValid)
			}
		})
	}
}
