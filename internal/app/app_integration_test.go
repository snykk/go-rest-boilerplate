//go:build integration

package app_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/snykk/go-rest-boilerplate/internal/app"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/contract"
	"github.com/snykk/go-rest-boilerplate/internal/modules/identity/usecase"
	"github.com/snykk/go-rest-boilerplate/internal/platform/id"
	"github.com/snykk/go-rest-boilerplate/internal/platform/lifecycle"
	"github.com/snykk/go-rest-boilerplate/internal/platform/logger"
	"github.com/snykk/go-rest-boilerplate/internal/platform/mailer"
	"github.com/snykk/go-rest-boilerplate/internal/platform/telemetry"
	"github.com/snykk/go-rest-boilerplate/internal/testenv"
	"github.com/snykk/go-rest-boilerplate/migrations"
)

const adminToken = "0123456789abcdef0123456789abcdef"

var (
	// codePattern finds the verification code in the text of its message.
	codePattern = regexp.MustCompile(`\b\d{6}\b`)
	// tokenPattern finds the password reset token, on a line of its own, in the text of its message.
	tokenPattern = regexp.MustCompile(`(?m)^[A-Za-z0-9_-]{43}$`)
)

// fixture is the application's handler on a database of its own, built as the server builds it except that its
// mail is recorded, with the logs and the log level for the test to read. Its identity module provisions the
// accounts no endpoint creates.
type fixture struct {
	config   app.Config
	deps     app.Deps
	logs     *bytes.Buffer
	level    *slog.LevelVar
	mail     *mailer.Recorder
	identity *identity.Module
	handler  http.Handler
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	pool := testenv.Postgres(t, migrations.FS)
	config, err := app.LoadConfig(env(map[string]string{
		"DATABASE_URL":         "postgres://unused", // the pool comes from testenv
		"ADMIN_TOKEN":          adminToken,
		"IDENTITY_JWT_SECRET":  jwtSecret,
		"IDENTITY_BCRYPT_COST": "10",
	}))
	if err != nil {
		t.Fatalf("LoadConfig = %v", err)
	}
	providers, err := telemetry.New(t.Context(), telemetry.Config{
		Service:     "test",
		SampleRatio: 1,
	})
	if err != nil {
		t.Fatalf("telemetry.New = %v", err)
	}

	f := &fixture{
		config: config,
		logs:   &bytes.Buffer{},
		level:  new(slog.LevelVar),
		mail:   &mailer.Recorder{},
	}
	log, err := logger.New(f.logs, logger.Options{
		Level:  f.level,
		Format: "json",
	})
	if err != nil {
		t.Fatalf("logger.New = %v", err)
	}
	f.deps = app.Deps{
		Log:       log,
		Level:     f.level,
		Pool:      pool,
		Telemetry: providers,
		Mail:      f.mail,
	}
	f.handler = newHandler(t, config, f.deps)
	f.identity = newIdentity(t, config, f.deps)
	return f
}

func newHandler(t *testing.T, config app.Config, deps app.Deps) http.Handler {
	t.Helper()
	handler, err := app.NewHandler(config, deps)
	if err != nil {
		t.Fatalf("NewHandler = %v", err)
	}
	return handler
}

func newIdentity(t *testing.T, config app.Config, deps app.Deps) *identity.Module {
	t.Helper()
	module, err := identity.New(identity.Deps{
		Pool:    deps.Pool,
		Log:     deps.Log,
		Mail:    deps.Mail,
		Service: config.Platform.Service,
		Config:  config.Identity,
	})
	if err != nil {
		t.Fatalf("identity.New = %v", err)
	}
	return module
}

// send sends one request with header to handler and records the response.
func send(handler http.Handler, method, target, body string, header map[string]string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	for key, value := range header {
		request.Header.Set(key, value)
	}
	handler.ServeHTTP(recorder, request)
	return recorder
}

// do sends one request to the fixture's handler, with token as its bearer token unless token is "".
func (f *fixture) do(method, target, body, token string) *httptest.ResponseRecorder {
	var header map[string]string
	if token != "" {
		header = map[string]string{"Authorization": "Bearer " + token}
	}
	return send(f.handler, method, target, body, header)
}

// provision creates a verified account with roles, as only an operator can, and returns its user ID.
func (f *fixture) provision(t *testing.T, email, password string, roles ...string) string {
	t.Helper()
	user, err := f.identity.Provision(t.Context(), usecase.ProvisionInput{
		Email:       email,
		Password:    password,
		DisplayName: "Test user",
		Roles:       roles,
	})
	if err != nil {
		t.Fatalf("Provision(%s) = %v", email, err)
	}
	return user.ID
}

// tokens is a token pair as the API returned it.
type tokens struct {
	access, refresh string
}

// signIn signs in through POST /v1/auth/login and returns the pair.
func (f *fixture) signIn(t *testing.T, email, password string) tokens {
	t.Helper()
	recorder := f.do("POST", "/v1/auth/login", credentials(email, password), "")
	wantStatus(t, recorder, 200, "sign in as "+email)
	return pairOf(t, recorder)
}

// mailed returns the text of the latest message of kind sent to email.
func (f *fixture) mailed(t *testing.T, email, kind string) string {
	t.Helper()
	messages := f.mail.Messages()
	for _, message := range slices.Backward(messages) {
		if message.To == email && message.Kind == kind {
			return message.Text
		}
	}
	t.Fatalf("no %s message to %s among %d", kind, email, len(messages))
	return ""
}

// logRecord returns the first line of logs that holds text, or "" when none does.
func logRecord(logs, text string) string {
	for line := range strings.Lines(logs) {
		if strings.Contains(line, text) {
			return line
		}
	}
	return ""
}

func credentials(email, password string) string {
	return `{"email":"` + email + `","password":"` + password + `"}`
}

func refreshBody(refreshToken string) string {
	return `{"refresh_token":"` + refreshToken + `"}`
}

func pairOf(t *testing.T, recorder *httptest.ResponseRecorder) tokens {
	t.Helper()
	access, _ := field(t, recorder, "access_token").(string)
	refresh, _ := field(t, recorder, "refresh_token").(string)
	if access == "" || refresh == "" || field(t, recorder, "token_type") != "Bearer" {
		t.Fatalf("body %s, want a Bearer token pair", recorder.Body)
	}
	return tokens{access: access, refresh: refresh}
}

func field(t *testing.T, recorder *httptest.ResponseRecorder, key string) any {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("body %s is not JSON: %v", recorder.Body, err)
	}
	return decoded[key]
}

func wantStatus(t *testing.T, recorder *httptest.ResponseRecorder, want int, step string) {
	t.Helper()
	if recorder.Code != want {
		t.Fatalf("%s: status = %d, want %d; body %s", step, recorder.Code, want, recorder.Body)
	}
}

func wantProblem(t *testing.T, recorder *httptest.ResponseRecorder, status int, code, step string) {
	t.Helper()
	wantStatus(t, recorder, status, step)
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/problem+json" ||
		field(t, recorder, "code") != code {
		t.Errorf("%s: %s %s, want an application/problem+json %s problem", step, contentType, recorder.Body, code)
	}
}

// NewHandler serves the application end to end on a real database: the article lifecycle for an editor, with
// changes refused to anyone else; the identity flows from registration to sign-out, concurrent refreshes and the
// reuse of a spent refresh token, password resets and lockouts; comments on published articles, which a reader
// cannot delete for another and an editor can; request IDs that follow the caller's trace; readiness; a Swagger
// UI that can load its scripts; and an operator endpoint for the log level that exists only with an admin token
// and requires it.
func TestNewHandler(t *testing.T) {
	f := newFixture(t)
	f.provision(t, "editor@example.com", "editor-password", contract.RoleReader, contract.RoleEditor)
	editor := f.signIn(t, "editor@example.com", "editor-password").access

	articleLifecycle(t, f, editor)
	t.Run("article changes need an editor", func(t *testing.T) { articleChangesNeedAnEditor(t, f) })
	t.Run("a user registers, verifies, signs in and out", func(t *testing.T) { signUpAndOut(t, f) })
	t.Run("an unverified user cannot sign in", func(t *testing.T) { unverifiedSignIn(t, f) })
	t.Run("one of twenty concurrent refreshes wins, and a reuse after the grace revokes them all",
		func(t *testing.T) { refreshRotation(t, f) })
	t.Run("a forgotten password is reset", func(t *testing.T) { passwordReset(t, f) })
	t.Run("a signed-in user changes the password", func(t *testing.T) { passwordChange(t, f) })
	t.Run("wrong passwords lock the account", func(t *testing.T) { lockout(t, f) })
	t.Run("once a concurrent attempt locks the account, the right password gets the answer of a wrong one",
		func(t *testing.T) { lockedMidway(t, f) })
	t.Run("a code or a reset token asked for again within the cooldown gets the same answer and no mail",
		func(t *testing.T) { resendCooldown(t, f) })
	t.Run("a code guessed at too often still holds the cooldown", func(t *testing.T) { exhaustedCode(t, f) })
	t.Run("an unknown email gets the same answer and no mail", func(t *testing.T) { unknownEmail(t, f) })
	t.Run("a reader comments on a published article and cannot delete another's comment, which an editor can",
		func(t *testing.T) { commenting(t, f, editor) })
	platformRoutes(t, f)
}

func articleLifecycle(t *testing.T, f *fixture, editor string) {
	recorder := f.do("POST", "/v1/articles", `{"slug":"hello-world","title":"Hello","body":"Text"}`, editor)
	wantStatus(t, recorder, 201, "create")
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Error("create: X-Request-ID is missing")
	}
	id := field(t, recorder, "id").(string)
	created := recorder.Body.String()
	if location := recorder.Header().Get("Location"); location != "/v1/articles/drafts/"+id {
		t.Errorf("create: Location = %q, want the draft's URL", location)
	}
	if got := f.do("GET", recorder.Header().Get("Location"), "", editor).Body.String(); got != created {
		t.Errorf("get the draft at Location = %s, want the article as create returned it: %s", got, created)
	}
	wantProblem(t, f.do("GET", "/v1/articles/"+id, "", ""), 404, "not_found", "a draft on the public routes")
	if total := field(t, f.do("GET", "/v1/articles?status=draft", "", ""), "total"); total != float64(0) {
		t.Errorf("public list with a draft stored: total = %v, want 0", total)
	}
	if total := field(t, f.do("GET", "/v1/articles/drafts", "", editor), "total"); total != float64(1) {
		t.Errorf("drafts list: total = %v, want 1", total)
	}

	wantStatus(t, f.do("POST", "/v1/articles", `{"slug":"hello-world","title":"Again"}`, editor), 409,
		"create with a used slug")
	wantProblem(t, f.do("POST", "/v1/articles", `{"slug":"nul","title":"t","body":"a\u0000b"}`, editor), 422,
		"invalid_body", "create with a NUL character in the body")
	wantProblem(t, f.do("GET", "/v1/nope", "", ""), 404, "not_found", "a path no route matches")
	recorder = f.do("PUT", "/v1/articles/"+id, "", editor)
	wantProblem(t, recorder, 405, "method_not_allowed", "a method the route does not take")
	if allow := recorder.Header().Get("Allow"); !strings.Contains(allow, "PATCH") {
		t.Errorf("a method the route does not take: Allow = %q, want it to list PATCH", allow)
	}

	recorder = f.do("POST", "/v1/articles/"+id+"/publish", "", editor)
	wantStatus(t, recorder, 200, "publish")
	if field(t, recorder, "status") != "published" || field(t, recorder, "published_at") == nil {
		t.Errorf("publish: body %s, want a published article with published_at", recorder.Body)
	}
	wantStatus(t, f.do("GET", "/v1/articles/"+id, "", ""), 200, "get after publish")
	wantProblem(t, f.do("GET", "/v1/articles/drafts/"+id, "", editor), 404, "not_found",
		"a published article on the drafts routes")

	wantStatus(t, f.do("PATCH", "/v1/articles/"+id, `{"slug":"renamed"}`, editor), 409, "rename a published slug")
	if total := field(t, f.do("GET", "/v1/articles", "", ""), "total"); total != float64(1) {
		t.Errorf("list published: total = %v, want 1", total)
	}
	wantStatus(t, f.do("DELETE", "/v1/articles/"+id, "", editor), 204, "delete")
	wantStatus(t, f.do("GET", "/v1/articles/"+id, "", ""), 404, "get after delete")
}

func articleChangesNeedAnEditor(t *testing.T, f *fixture) {
	const article = `{"slug":"by-a-reader","title":"Mine"}`
	f.provision(t, "reader@example.com", "reader-password", contract.RoleReader)
	reader := f.signIn(t, "reader@example.com", "reader-password").access

	recorder := f.do("POST", "/v1/articles", article, "")
	wantProblem(t, recorder, 401, "unauthenticated", "create without a token")
	if challenge := recorder.Header().Get("WWW-Authenticate"); challenge != "Bearer" {
		t.Errorf("create without a token: WWW-Authenticate = %q, want Bearer", challenge)
	}
	wantProblem(t, f.do("POST", "/v1/articles", article, "not-a-token"), 401, "unauthenticated",
		"create with a token that is not one")
	wantProblem(t, f.do("POST", "/v1/articles", article, reader), 403, "forbidden", "create as a reader")
	wantProblem(t, f.do("GET", "/v1/articles/drafts", "", reader), 403, "forbidden", "list drafts as a reader")
}

func signUpAndOut(t *testing.T, f *fixture) {
	const email, password = "ada@example.com", "correct-horse-1"
	recorder := f.do("POST", "/v1/auth/register",
		`{"email":"`+email+`","password":"`+password+`","display_name":"Ada"}`, "")
	wantStatus(t, recorder, 201, "register")
	if field(t, recorder, "email") != email || field(t, recorder, "verified") != false {
		t.Errorf("register: body %s, want an unverified %s", recorder.Body, email)
	}
	wantProblem(t, f.do("POST", "/v1/auth/register",
		`{"email":"ADA@example.com","password":"`+password+`","display_name":"Twin"}`, ""), 409, "email_taken",
		"register an email in another case")

	code := codePattern.FindString(f.mailed(t, email, "identity.verification"))
	wantStatus(t, f.do("POST", "/v1/auth/verification", `{"email":"`+email+`","code":"`+code+`"}`, ""), 204,
		"verify with the mailed code")
	signedIn := f.signIn(t, email, password)

	recorder = f.do("GET", "/v1/users/me", "", signedIn.access)
	wantStatus(t, recorder, 200, "profile")
	if field(t, recorder, "email") != email || field(t, recorder, "verified") != true {
		t.Errorf("profile: body %s, want a verified %s", recorder.Body, email)
	}
	wantProblem(t, f.do("GET", "/v1/users/me", "", ""), 401, "unauthenticated", "profile without a token")

	recorder = f.do("POST", "/v1/auth/refresh", refreshBody(signedIn.refresh), "")
	wantStatus(t, recorder, 200, "refresh")
	refreshed := pairOf(t, recorder)
	wantStatus(t, f.do("POST", "/v1/auth/logout", refreshBody(refreshed.refresh), ""), 204, "logout")
	wantProblem(t, f.do("POST", "/v1/auth/refresh", refreshBody(refreshed.refresh), ""), 401, "invalid_token",
		"refresh after logout")
}

func unverifiedSignIn(t *testing.T, f *fixture) {
	const email, password = "bob@example.com", "correct-horse-2"
	wantStatus(t, f.do("POST", "/v1/auth/register",
		`{"email":"`+email+`","password":"`+password+`","display_name":"Bob"}`, ""), 201, "register")
	wantProblem(t, f.do("POST", "/v1/auth/login", credentials(email, password), ""), 403, "email_not_verified",
		"sign in before verifying")

	code := codePattern.FindString(f.mailed(t, email, "identity.verification"))
	wrong := `{"email":"` + email + `","code":"` + otherCode(code) + `"}`
	for attempt := range 5 {
		wantProblem(t, f.do("POST", "/v1/auth/verification", wrong, ""), 422, "invalid_code",
			fmt.Sprintf("wrong code %d", attempt+1))
	}
	wantProblem(t, f.do("POST", "/v1/auth/verification", `{"email":"`+email+`","code":"`+code+`"}`, ""), 422,
		"too_many_attempts", "the right code after five wrong ones")
}

// otherCode returns a six-digit code that is not code.
func otherCode(code string) string {
	if code == "000000" {
		return "000001"
	}
	return "000000"
}

func refreshRotation(t *testing.T, f *fixture) {
	const email = "carol@example.com"
	userID := f.provision(t, email, "correct-horse-3", contract.RoleReader)
	signedIn := f.signIn(t, email, "correct-horse-3")

	recorders := make([]*httptest.ResponseRecorder, 20)
	var refreshing sync.WaitGroup
	for i := range recorders {
		refreshing.Go(func() { recorders[i] = f.do("POST", "/v1/auth/refresh", refreshBody(signedIn.refresh), "") })
	}
	refreshing.Wait()

	var winners []*httptest.ResponseRecorder
	for _, recorder := range recorders {
		switch recorder.Code {
		case 200:
			winners = append(winners, recorder)
		case 401:
			wantProblem(t, recorder, 401, "invalid_token", "a refresh that lost")
		default:
			t.Errorf("a concurrent refresh: status = %d, want 200 or 401; body %s", recorder.Code, recorder.Body)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("concurrent refreshes: %d answered 200, want exactly 1", len(winners))
	}
	recorder := f.do("POST", "/v1/auth/refresh", refreshBody(pairOf(t, winners[0]).refresh), "")
	wantStatus(t, recorder, 200, "refresh with the winner's token")
	newest := pairOf(t, recorder)

	config := f.config
	config.Identity.RefreshReuseGrace = time.Millisecond
	shortGrace := newHandler(t, config, f.deps)
	time.Sleep(10 * config.Identity.RefreshReuseGrace)
	wantProblem(t, send(shortGrace, "POST", "/v1/auth/refresh", refreshBody(signedIn.refresh), nil), 401,
		"invalid_token", "the first token again after the grace")
	wantProblem(t, f.do("POST", "/v1/auth/refresh", refreshBody(newest.refresh), ""), 401, "invalid_token",
		"the winner's newest token after the reuse")

	reuse := logRecord(f.logs.String(), `"event":"refresh_reused"`)
	if !strings.Contains(reuse, `"user_id":"`+userID+`"`) || strings.Contains(f.logs.String(), email) {
		t.Errorf("reuse: security event = %q, email logged = %v; want refresh_reused with user_id %s, false",
			reuse, strings.Contains(f.logs.String(), email), userID)
	}
}

func passwordReset(t *testing.T, f *fixture) {
	const email = "dan@example.com"
	f.provision(t, email, "old-password-1", contract.RoleReader)
	before := f.signIn(t, email, "old-password-1")

	wantStatus(t, f.do("POST", "/v1/auth/password/forgot", `{"email":"`+email+`"}`, ""), 202, "forgot")
	token := tokenPattern.FindString(f.mailed(t, email, "identity.password_reset"))
	reset := `{"token":"` + token + `","password":"new-password-1"}`
	wantStatus(t, f.do("POST", "/v1/auth/password/reset", reset, ""), 204, "reset with the mailed token")
	wantProblem(t, f.do("POST", "/v1/auth/password/reset", reset, ""), 422, "invalid_token", "reset twice")

	wantProblem(t, f.do("POST", "/v1/auth/refresh", refreshBody(before.refresh), ""), 401, "invalid_token",
		"refresh with a token from before the reset")
	wantProblem(t, f.do("POST", "/v1/auth/login", credentials(email, "old-password-1"), ""), 401,
		"invalid_credentials", "sign in with the old password")
	f.signIn(t, email, "new-password-1")
}

func passwordChange(t *testing.T, f *fixture) {
	const email = "erin@example.com"
	f.provision(t, email, "old-password-2", contract.RoleReader)
	before := f.signIn(t, email, "old-password-2")

	wantProblem(t, f.do("POST", "/v1/auth/password/change",
		`{"current_password":"wrong-password","new_password":"new-password-2"}`, before.access), 401,
		"invalid_credentials", "change with a wrong current password")
	recorder := f.do("POST", "/v1/auth/password/change",
		`{"current_password":"old-password-2","new_password":"new-password-2"}`, before.access)
	wantStatus(t, recorder, 200, "change")
	after := pairOf(t, recorder)

	wantProblem(t, f.do("POST", "/v1/auth/refresh", refreshBody(before.refresh), ""), 401, "invalid_token",
		"refresh with a token from before the change")
	wantStatus(t, f.do("POST", "/v1/auth/refresh", refreshBody(after.refresh), ""), 200,
		"refresh with the token of the change")
	f.signIn(t, email, "new-password-2")
}

func lockout(t *testing.T, f *fixture) {
	const email = "frank@example.com"
	f.provision(t, email, "right-password", contract.RoleReader)

	for attempt := range 5 {
		wantProblem(t, f.do("POST", "/v1/auth/login", credentials(email, "wrong-password"), ""), 401,
			"invalid_credentials", fmt.Sprintf("wrong password %d", attempt+1))
	}
	wantProblem(t, f.do("POST", "/v1/auth/login", credentials(email, "right-password"), ""), 423, "account_locked",
		"the right password after five wrong ones")

	logs := f.logs.String()
	locked := strings.Contains(logs, `"event":"account_locked"`)
	if leaked := strings.Contains(logs, email) || strings.Contains(logs, "wrong-password"); !locked || leaked {
		t.Errorf("lockout: an account_locked security event logged = %v, the email or a password logged = %v; "+
			"want true, false", locked, leaked)
	}
}

// lockedMidway lets three wrong passwords and the right one read an account and compare their passwords while
// a transaction, standing in for the failure that takes the lock in a burst of guesses, holds the account's row
// and locks it. Once that transaction commits, every one of them must answer as a locked account does, so no
// answer tells the right password apart.
func lockedMidway(t *testing.T, f *fixture) {
	const email, password = "heidi@example.com", "right-password"
	userID := f.provision(t, email, password, contract.RoleReader)
	attempts := []string{"wrong-password-1", "wrong-password-2", "wrong-password-3", password}
	if conns := f.deps.Pool.Config().MaxConns; conns < int32(len(attempts)) {
		t.Fatalf("the pool holds %d connections, want at least %d, one per waiting sign-in", conns, len(attempts))
	}

	locker, err := f.connect(t).Begin(t.Context())
	if err != nil {
		t.Fatalf("Begin = %v", err)
	}
	defer func() { _ = locker.Rollback(context.Background()) }()
	_, err = locker.Exec(t.Context(), `UPDATE users SET failed_logins = $2, locked_until = $3 WHERE id = $1`,
		userID, f.config.Identity.LoginAttempts, time.Now().Add(f.config.Identity.Lockout))
	if err != nil {
		t.Fatalf("lock the account = %v", err)
	}

	recorders := make([]*httptest.ResponseRecorder, len(attempts))
	var signingIn sync.WaitGroup
	for i, attempt := range attempts {
		signingIn.Go(func() { recorders[i] = f.do("POST", "/v1/auth/login", credentials(email, attempt), "") })
	}
	waitForLockWaiters(t, f.connect(t), len(attempts))
	if err := locker.Commit(t.Context()); err != nil {
		t.Fatalf("Commit = %v", err)
	}
	signingIn.Wait()

	for i, recorder := range recorders {
		wantProblem(t, recorder, 423, "account_locked", fmt.Sprintf("sign-in %d of %d, the last with the right "+
			"password, read before the lock", i+1, len(attempts)))
	}
}

// connect opens a connection to the fixture's database outside its pool, closed when the test ends.
func (f *fixture) connect(t *testing.T) *pgx.Conn {
	t.Helper()
	connection, err := pgx.Connect(t.Context(), f.deps.Pool.Config().ConnString())
	if err != nil {
		t.Fatalf("connect = %v", err)
	}
	t.Cleanup(func() { _ = connection.Close(context.Background()) })
	return connection
}

// waitForLockWaiters returns once count queries on watcher's database wait for a lock, and fails the test when
// fewer do after ten seconds.
func waitForLockWaiters(t *testing.T, watcher *pgx.Conn, count int) {
	t.Helper()
	var waiting int
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); {
		err := watcher.QueryRow(t.Context(),
			`SELECT count(*) FROM pg_stat_activity
			  WHERE datname = current_database() AND wait_event_type = 'Lock'`).Scan(&waiting)
		if err != nil {
			t.Fatalf("select lock waiters = %v", err)
		}
		if waiting >= count {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("queries waiting for a lock after ten seconds = %d, want %d", waiting, count)
}

// resendCooldown asks for a verification code right after registering, and for a password reset twice in a
// row. A request within IDENTITY_RESEND_COOLDOWN answers 202 like the others but mails nothing, and the code or
// token mailed before it still works.
func resendCooldown(t *testing.T, f *fixture) {
	const email, password = "ivan@example.com", "correct-horse-4"
	wantStatus(t, f.do("POST", "/v1/auth/register",
		`{"email":"`+email+`","password":"`+password+`","display_name":"Ivan"}`, ""), 201, "register")
	code := codePattern.FindString(f.mailed(t, email, "identity.verification"))
	before := len(f.mail.Messages())

	wantStatus(t, f.do("POST", "/v1/auth/verification/resend", `{"email":"`+email+`"}`, ""), 202,
		"resend right after registering")
	wantStatus(t, f.do("POST", "/v1/auth/verification", `{"email":"`+email+`","code":"`+code+`"}`, ""), 204,
		"verify with the code mailed at registration")
	wantStatus(t, f.do("POST", "/v1/auth/password/forgot", `{"email":"`+email+`"}`, ""), 202, "forgot")
	token := tokenPattern.FindString(f.mailed(t, email, "identity.password_reset"))
	wantStatus(t, f.do("POST", "/v1/auth/password/forgot", `{"email":"`+email+`"}`, ""), 202,
		"forgot again within the cooldown")

	if after := len(f.mail.Messages()); after != before+1 {
		t.Errorf("messages after a resend and two forgots within the cooldown = %d, want %d, the first reset only",
			after, before+1)
	}
	wantStatus(t, f.do("POST", "/v1/auth/password/reset", `{"token":"`+token+`","password":"new-password-4"}`, ""),
		204, "reset with the first token")
}

// exhaustedCode guesses a verification code wrong six times, which exhausts it, and asks for a new one within
// IDENTITY_RESEND_COOLDOWN. The resend answers 202 but mails nothing, and the right code stays refused.
func exhaustedCode(t *testing.T, f *fixture) {
	const email, password = "judy@example.com", "correct-horse-5"
	wantStatus(t, f.do("POST", "/v1/auth/register",
		`{"email":"`+email+`","password":"`+password+`","display_name":"Judy"}`, ""), 201, "register")
	code := codePattern.FindString(f.mailed(t, email, "identity.verification"))
	wrong := `{"email":"` + email + `","code":"` + otherCode(code) + `"}`
	for attempt := range 5 {
		wantProblem(t, f.do("POST", "/v1/auth/verification", wrong, ""), 422, "invalid_code",
			fmt.Sprintf("wrong code %d", attempt+1))
	}
	wantProblem(t, f.do("POST", "/v1/auth/verification", wrong, ""), 422, "too_many_attempts", "wrong code 6")
	before := len(f.mail.Messages())

	wantStatus(t, f.do("POST", "/v1/auth/verification/resend", `{"email":"`+email+`"}`, ""), 202,
		"resend within the cooldown of the exhausted code")

	if after := len(f.mail.Messages()); after != before {
		t.Errorf("messages after the resend = %d, want %d", after, before)
	}
	wantProblem(t, f.do("POST", "/v1/auth/verification", `{"email":"`+email+`","code":"`+code+`"}`, ""), 422,
		"too_many_attempts", "the right code after the resend")
}

func unknownEmail(t *testing.T, f *fixture) {
	before := len(f.mail.Messages())

	wantStatus(t, f.do("POST", "/v1/auth/password/forgot", `{"email":"nobody@example.com"}`, ""), 202,
		"forgot for an unknown email")
	wantStatus(t, f.do("POST", "/v1/auth/verification/resend", `{"email":"nobody@example.com"}`, ""), 202,
		"resend for an unknown email")

	if after := len(f.mail.Messages()); after != before {
		t.Errorf("messages after requests for an unknown email = %d, want %d", after, before)
	}
}

func commenting(t *testing.T, f *fixture, editor string) {
	const email, password = "grace@example.com", "reader-password"
	readerID := f.provision(t, email, password, contract.RoleReader)
	reader := f.signIn(t, email, password).access
	articleID := f.publish(t, editor, "commented")
	recorder := f.do("POST", "/v1/articles", `{"slug":"not-yet","title":"Draft"}`, editor)
	wantStatus(t, recorder, 201, "create a draft")
	draftID := field(t, recorder, "id").(string)

	recorder = f.do("POST", "/v1/comments", commentBody(articleID, "First!"), reader)
	wantStatus(t, recorder, 201, "a reader comments")
	if field(t, recorder, "author_id") != readerID || field(t, recorder, "author_name") != "Test user" {
		t.Errorf("a reader comments: body %s, want the reader as the author, with their display name", recorder.Body)
	}
	byReader := field(t, recorder, "id").(string)
	recorder = f.do("POST", "/v1/comments", commentBody(articleID, "Thanks for reading."), editor)
	wantStatus(t, recorder, 201, "the editor comments")
	byEditor := field(t, recorder, "id").(string)

	wantProblem(t, f.do("POST", "/v1/comments", commentBody(articleID, "Anonymous"), ""), 401, "unauthenticated",
		"comment without a token")
	wantProblem(t, f.do("POST", "/v1/comments", commentBody(draftID, "Too early"), reader), 404, "not_found",
		"comment on a draft")
	wantProblem(t, f.do("POST", "/v1/comments", commentBody("commented", "Wrong ID"), reader), 422,
		"validation_failed", "comment on an article ID that is not a UUID")
	wantProblem(t, f.do("PATCH", "/v1/articles/"+articleID, `{"title":"Mine now"}`, reader), 403, "forbidden",
		"a reader changes the article")

	list := "/v1/comments?article_id=" + articleID
	if got, want := commentIDs(t, f.do("GET", list, "", "")), []string{byEditor, byReader}; !slices.Equal(got, want) {
		t.Errorf("anyone lists the comments: %v, want %v, newest first", got, want)
	}
	byAuthor := list + "&author_id=" + readerID
	if got, want := commentIDs(t, f.do("GET", byAuthor, "", "")), []string{byReader}; !slices.Equal(got, want) {
		t.Errorf("list the reader's comments: %v, want %v", got, want)
	}
	wantProblem(t, f.do("GET", "/v1/comments?article_id="+draftID, "", ""), 404, "not_found",
		"list the comments of a draft")
	wantProblem(t, f.do("GET", "/v1/comments?article_id="+id.New(), "", ""), 404, "not_found",
		"list the comments of an article that never existed")
	wantProblem(t, f.do("GET", "/v1/comments", "", ""), 400, "invalid_query", "list without an article")

	wantProblem(t, f.do("DELETE", "/v1/comments/"+byEditor, "", reader), 403, "forbidden",
		"the reader deletes the editor's comment")
	wantStatus(t, f.do("DELETE", "/v1/comments/"+byReader, "", editor), 204, "the editor deletes the reader's comment")
	wantProblem(t, f.do("DELETE", "/v1/comments/"+byReader, "", reader), 404, "not_found",
		"the reader deletes the comment the editor deleted")
	if got, want := commentIDs(t, f.do("GET", list, "", "")), []string{byEditor}; !slices.Equal(got, want) {
		t.Errorf("list after the deletion: %v, want %v", got, want)
	}
}

// publish creates an article with slug as editor and publishes it, and returns its ID.
func (f *fixture) publish(t *testing.T, editor, slug string) string {
	t.Helper()
	recorder := f.do("POST", "/v1/articles", `{"slug":"`+slug+`","title":"Title","body":"Text"}`, editor)
	wantStatus(t, recorder, 201, "create "+slug)
	articleID := field(t, recorder, "id").(string)
	wantStatus(t, f.do("POST", "/v1/articles/"+articleID+"/publish", "", editor), 200, "publish "+slug)
	return articleID
}

func commentBody(articleID, text string) string {
	return `{"article_id":"` + articleID + `","body":"` + text + `"}`
}

// commentIDs returns the IDs of the comments a list answered with, in order. It fails the test unless the list
// answered 200.
func commentIDs(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()
	wantStatus(t, recorder, 200, "list comments")
	var page struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &page); err != nil {
		t.Fatalf("body %s is not a list of comments: %v", recorder.Body, err)
	}
	ids := make([]string, len(page.Items))
	for i, item := range page.Items {
		ids[i] = item.ID
	}
	return ids
}

func platformRoutes(t *testing.T, f *fixture) {
	wantStatus(t, f.do("GET", "/ready", "", ""), 200, "ready")

	recorder := f.do("GET", "/swagger/index.html", "", "")
	wantStatus(t, recorder, 200, "swagger UI")
	if csp := recorder.Header().Get("Content-Security-Policy"); !strings.Contains(csp, "script-src 'self'") {
		t.Errorf("swagger UI CSP = %q, want scripts allowed from 'self'", csp)
	}
	wantStatus(t, f.do("GET", "/swagger/doc.json", "", ""), 200, "swagger spec")

	const callerTrace = "0af7651916cd43dd8448eb211c80319c"
	f.logs.Reset()
	recorder = send(f.handler, "GET", "/v1/articles", "",
		map[string]string{"traceparent": "00-" + callerTrace + "-b7ad6b7169203331-01"})
	wantStatus(t, recorder, 200, "list in the caller's trace")
	if requestID := recorder.Header().Get("X-Request-ID"); requestID != callerTrace {
		t.Errorf("list in the caller's trace: X-Request-ID = %q, want the trace ID %q", requestID, callerTrace)
	}
	if !strings.Contains(f.logs.String(), `"trace_id":"`+callerTrace+`"`) {
		t.Errorf("list in the caller's trace: logs %s, want the trace ID in the access log", f.logs.String())
	}

	wantProblem(t, f.do("PUT", "/admin/log-level", `{"level":"debug"}`, ""), 401, "unauthenticated",
		"log level without the token")
	wantStatus(t, f.do("PUT", "/admin/log-level", `{"level":"debug"}`, adminToken), 200, "log level with the token")
	if f.level.Level() != slog.LevelDebug {
		t.Errorf("log level with the token: level = %s, want DEBUG", f.level.Level())
	}

	config := f.config
	config.Platform.Admin.Token = ""
	withoutAdmin := newHandler(t, config, f.deps)
	wantStatus(t, send(withoutAdmin, "GET", "/admin/log-level", "", nil), 404,
		"log level without an admin token configured")
}

// App.Run gives the components the HTTP shutdown timeout and a moment more to stop, so a component that stops on
// that deadline, as the HTTP server does when requests outlive it, reports its own error rather than a timeout.
// A component that does not stop is reported with lifecycle.ErrStopTimeout. Run closes the resources after the
// components.
func TestAppRun(t *testing.T) {
	const shutdown = 300 * time.Millisecond
	errLateStop := errors.New("stopped on the shutdown deadline")
	tCases := []struct {
		name    string
		stop    time.Duration // how long the component takes to return once its context ends
		never   bool          // the component does not return
		result  error         // what the component returns
		wantErr error         // nil means Run returns nil
		notErr  error         // an error Run must not report
	}{
		{name: "a component that stops in time gives nil"},
		{
			name:    "a component that stops on the shutdown deadline reports its own error, not a timeout",
			stop:    shutdown + 50*time.Millisecond,
			result:  errLateStop,
			wantErr: errLateStop,
			notErr:  lifecycle.ErrStopTimeout,
		},
		{
			name:    "a component that does not stop is reported as late",
			never:   true,
			wantErr: lifecycle.ErrStopTimeout,
		},
	}

	for _, tc := range tCases {
		t.Run(tc.name, func(t *testing.T) {
			config, err := app.LoadConfig(env(map[string]string{
				"DATABASE_URL":          testenv.Postgres(t, nil).Config().ConnString(),
				"HTTP_SHUTDOWN_TIMEOUT": shutdown.String(),
				"IDENTITY_JWT_SECRET":   jwtSecret,
				"IDENTITY_BCRYPT_COST":  "10",
			}))
			if err != nil {
				t.Fatalf("LoadConfig = %v", err)
			}
			application, err := app.New(t.Context(), config)
			if err != nil {
				t.Fatalf("New = %v", err)
			}
			release := make(chan struct{})
			t.Cleanup(func() { close(release) })
			component := lifecycle.ComponentFunc(func(ctx context.Context) error {
				<-ctx.Done()
				if tc.never {
					<-release
				}
				time.Sleep(tc.stop)
				return tc.result
			})
			ctx, cancel := context.WithCancel(t.Context())
			cancel()

			err = application.Run(ctx, component)

			if !errors.Is(err, tc.wantErr) || (tc.wantErr == nil && err != nil) {
				t.Errorf("Run = %v, want %v", err, tc.wantErr)
			}
			if tc.notErr != nil && errors.Is(err, tc.notErr) {
				t.Errorf("Run = %v, want no %v", err, tc.notErr)
			}
		})
	}
}
