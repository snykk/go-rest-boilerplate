//go:build integration

package main

import (
	"testing"

	"github.com/snykk/go-rest-boilerplate/internal/testenv"
)

func TestMain(m *testing.M) { testenv.Main(m) }
