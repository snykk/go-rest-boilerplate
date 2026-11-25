//go:build integration

package postgres_test

import (
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }
