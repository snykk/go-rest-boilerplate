package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
)

// client sends requests to the application's handler in process, as a caller of the API would, so that every
// middleware and rule applies.
type client struct {
	handler http.Handler
}

// problemError is the answer of the API when its status is not the one the caller expected: the status and the
// code of the problem document, never its text.
type problemError struct {
	status int
	code   string
}

func (e *problemError) Error() string {
	return fmt.Sprintf("status %d, problem %q", e.status, e.code)
}

// hasProblem reports whether err is a problemError with the given code.
func hasProblem(err error, code string) bool {
	problem, ok := errors.AsType[*problemError](err)
	return ok && problem.code == code
}

// do sends one request, with token as its bearer token unless token is "" and with payload as its JSON body
// unless payload is nil. When the status is want it decodes the body into result unless result is nil, and
// otherwise it returns a *problemError.
func (c client) do(ctx context.Context, token, method, target string, payload, result any, want int) error {
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(encoded)
	}
	request := httptest.NewRequestWithContext(ctx, method, target, body)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	c.handler.ServeHTTP(recorder, request)

	if recorder.Code != want {
		var problem struct {
			Code string `json:"code"`
		}
		_ = json.Unmarshal(recorder.Body.Bytes(), &problem)
		return &problemError{status: recorder.Code, code: problem.Code}
	}
	if result == nil {
		return nil
	}
	return json.Unmarshal(recorder.Body.Bytes(), result)
}

// signIn signs in through POST /v1/auth/login and returns the access token.
func (c client) signIn(ctx context.Context, email, password string) (string, error) {
	var pair struct {
		AccessToken string `json:"access_token"`
	}
	credentials := map[string]string{"email": email, "password": password}
	if err := c.do(ctx, "", http.MethodPost, "/v1/auth/login", credentials, &pair, http.StatusOK); err != nil {
		return "", err
	}
	return pair.AccessToken, nil
}
