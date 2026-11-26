# Module: identity

Users and how they sign in. A user registers with an email, a password and a display name, verifies the email
with a mailed code, and signs in for a short-lived access token and a refresh token that is replaced at each
use. A forgotten password is reset with a mailed token; a signed-in user changes it by giving the current one.

## Endpoints

All under `/v1`; errors are RFC 9457 problems. `RegisterRoutes(public, signedIn)` puts the endpoints that serve
anyone on `public` and the two that act for the caller on `signedIn`, which `internal/app` builds with
`auth.Authenticate` over a `jwt.Verifier` of the same settings.

| Group | Method | Path | Success | Errors |
|---|---|---|---|---|
| public | POST | `/auth/register` | 201, the user, unverified, with the role `reader`; the code is mailed | 409 `email_taken`, 422 `validation_failed` |
| public | POST | `/auth/verification/resend` | 202, no body, whatever the email | 422 `validation_failed` |
| public | POST | `/auth/verification` | 204 | 422 `validation_failed`, `invalid_code`, `code_expired`, `too_many_attempts` |
| public | POST | `/auth/login` | 200, a token pair | 401 `invalid_credentials`, 403 `email_not_verified`, 422 `validation_failed`, 423 `account_locked` |
| public | POST | `/auth/refresh` | 200, the next token pair | 401 `invalid_token`, 422 `validation_failed` |
| public | POST | `/auth/logout` | 204, also for an unknown token | 422 `validation_failed` |
| public | POST | `/auth/password/forgot` | 202, no body, whatever the email | 422 `validation_failed` |
| public | POST | `/auth/password/reset` | 204 | 422 `validation_failed`, `invalid_token` |
| signedIn | POST | `/auth/password/change` | 200, the token pair of a new sign-in | 401 `unauthenticated`, `invalid_credentials`, 404 `not_found`, 422 `validation_failed`, 423 `account_locked` |
| signedIn | GET | `/users/me` | 200, the caller | 401 `unauthenticated`, 404 `not_found` |

Every endpoint with a body also answers 400 `malformed_json` and 413 `body_too_large`. A token pair is
`{access_token, access_expires_at, refresh_token, refresh_expires_at, token_type: "Bearer"}`, sent with
`Cache-Control: no-store`. A value the rules refuse is a `validation_failed` problem that names its field
(`email`, `password`, `display_name`, or `new_password` for a change) and never repeats the value.

## Settings

Read by `config.go`; the application reports an invalid value at start-up.

| Variable | Default | Notes |
|---|---|---|
| `IDENTITY_JWT_SECRET` | required | Signs and verifies access tokens (HS256); at least 32 bytes. `internal/app` refuses the development value of `.env.example` in production |
| `IDENTITY_JWT_ISSUER` | `go-rest-boilerplate` | The `iss` claim |
| `IDENTITY_JWT_AUDIENCE` | `go-rest-boilerplate` | The `aud` claim |
| `IDENTITY_ACCESS_TTL` | `15m` | Lifetime of an access token; at least `1m` |
| `IDENTITY_REFRESH_TTL` | `720h` | Lifetime of a refresh token; at least `1m` |
| `IDENTITY_REFRESH_REUSE_GRACE` | `10s` | How long after a refresh a client may present the spent token again without it being taken as stolen |
| `IDENTITY_CODE_TTL` | `10m` | Lifetime of a verification code; at least `1m` |
| `IDENTITY_CODE_ATTEMPTS` | `5` | Wrong guesses that end a verification code |
| `IDENTITY_RESET_TTL` | `30m` | Lifetime of a password reset token; at least `1m` |
| `IDENTITY_RESEND_COOLDOWN` | `60s` | How long after a code or a reset token was issued to a user `resend` and `forgot` issue and mail nothing for them |
| `IDENTITY_BCRYPT_COST` | `12` | From 10 to 14 |
| `IDENTITY_LOGIN_ATTEMPTS` | `5` | Failed sign-ins in a row that lock an account |
| `IDENTITY_LOCKOUT` | `15m` | How long a locked account stays locked |

The verifier accepts an access token up to 10 seconds past its expiry, for clock drift between replicas.

## Rules

- `email`: trimmed and lower-cased, at most 254 bytes, one `@` between a non-empty local part and domain, and
  no whitespace, quote, angle bracket or control character. One account per email, in any case.
- `password`: 8 to 72 bytes (bcrypt reads no more), stored as a bcrypt hash.
- `display_name`: trimmed, 1 to 50 characters, none of them a control character.
- Roles: `reader` for every user who registers (`contract.RoleReader`). No endpoint grants a role. `Provision`
  creates a verified account with the roles it is given, such as `contract.RoleEditor`, and only `cmd/seed`
  calls it. In production an operator grants `editor` to a user who has verified their email with one SQL
  statement on `users`, as "Editors in production" in the repository README shows; the new role is in the access
  token issued at the user's next sign-in or refresh.
- A user signs in once the email is verified. The code is six random digits, mailed, stored as a SHA-256 hash,
  and ends when it is used, expires, or has been guessed at `IDENTITY_CODE_ATTEMPTS` times. A new one replaces
  the old.
- Failed sign-ins in a row lock the account for `IDENTITY_LOCKOUT`. A locked account is refused before the
  password is compared, and the lock is checked again under the user's row lock when an attempt is saved: an
  attempt that read the account before a concurrent one locked it answers `account_locked` too, with a wrong
  password or the right one, verified or not, so once the lock is taken no answer to a burst of guesses tells
  the right password apart. A wrong current password in a change counts as a failed sign-in, and a right one is
  checked against the lock under the row lock before the new password is hashed, so after the lock both answer
  `account_locked` as fast. A password reset clears the count and ends the lockout, since it proves the user
  holds the mailbox. An unknown email is
  compared with a dummy hash of the same cost, so it takes as long as a wrong password, and counts against no
  account.
- Each sign-in starts a family of refresh tokens. A refresh spends the token and issues the next one of the
  family, in one conditional update: of concurrent refreshes with one token, exactly one wins. A token spent
  less than `IDENTITY_REFRESH_REUSE_GRACE` ago is refused and changes nothing, so two tabs refreshing at once
  stay signed in; one spent earlier is taken as stolen, and its whole family is revoked.
- A sign-out revokes the family of its token. A reset revokes every session of the user, in one unit of work
  with the new password and the spent token; a change does the same and starts a new family.
- Access tokens are not checked against the store: one stays valid until it expires after a sign-out, a reset
  or a change. `IDENTITY_ACCESS_TTL` is the lever.
- `forgot` and `resend` answer the same for any email, so the answer does not tell whether an account exists,
  and their mail goes out in the background rather than during the request. A reset token is 32 random bytes
  in base64url, stored as a SHA-256 hash, single use.
- For `IDENTITY_RESEND_COOLDOWN` after a code or a reset token was issued to a user, at registration or by a
  request, `resend` and `forgot` keep it and issue and mail nothing for that user, with the same answer. So a
  new code, with a new budget of `IDENTITY_CODE_ATTEMPTS` guesses, comes at most once per cooldown, and nobody
  can flood an inbox. The check and the write are one statement, so a burst of requests issues one secret. A
  code or a token refused as expired or as guessed at too often stays in its table, still refused, until a new
  one replaces it, so the cooldown counts from it too: exhausting a code does not bring the next one sooner.
- No problem or log record carries a password, a code or a token; logs name a message by its kind only.
- Security events are log records, not a separate audit stream: the message `security event` with the
  attributes `event` and `user_id`, never an email or a submitted value. The use cases log them once the write
  they report has succeeded:

  | `event` | Level | When |
  |---|---|---|
  | `login_succeeded` | INFO | a sign-in's session is stored |
  | `login_failed` | WARN | a sign-in, or a password change, is refused as `invalid_credentials`, or has a wrong password and finds the account locked by a concurrent attempt; a sign-in refused before a user is known, because the email is invalid, the password is outside 8 to 72 bytes or no user has the email, has no `user_id` |
  | `account_locked` | WARN | the failure is the one that locks the account; once per lock |
  | `refresh_reused` | WARN | a refresh token spent earlier than the grace came back, and its family was revoked |
  | `password_changed` | INFO | a password change commits |
  | `password_reset` | INFO | a password reset commits |

## Tables

`users`

| Column | Type | Notes |
|---|---|---|
| `id` | `uuid` | primary key, UUIDv7 |
| `email` | `text` | unique index `users_email_key` on `lower(email)` |
| `display_name` | `text` | |
| `password_hash` | `text` | bcrypt |
| `roles` | `text[]` | |
| `verified_at` | `timestamptz` | null until the email is verified |
| `failed_logins` | `integer` | failed sign-ins since the last successful one |
| `locked_until` | `timestamptz` | the end of the latest lockout, or null |
| `created_at`, `updated_at` | `timestamptz` | |

`sessions`: one row per refresh token: `id`, `user_id` (cascades from `users`), `family`, `token_hash` (unique,
SHA-256), `expires_at`, `spent_at`, `revoked_at`, `created_at`; indexes on `user_id` and `family`.

`email_verifications` and `password_resets`: at most one secret per user: `user_id` (primary key, cascades
from `users`), `hash` (SHA-256; unique for reset tokens, not for six-digit codes), `expires_at`, `attempts`,
`created_at` (when it was issued, which the cooldown counts from).

The foreign keys stay inside the module.

Nothing deletes old rows. Every refresh adds a `sessions` row, and spent, revoked and expired ones stay, as do
expired codes and reset tokens until a new one replaces them, so these tables grow with use. An operator can
run this cleanup; it keeps every session of a family that still has a live token, so a spent token presented
again still revokes its family, and keeps a secret a day past its expiry, so the cooldown still counts from it:

```sql
DELETE FROM sessions s
 WHERE s.expires_at < now()
   AND NOT EXISTS (SELECT 1 FROM sessions f
                    WHERE f.family = s.family AND f.revoked_at IS NULL AND f.expires_at > now());
DELETE FROM email_verifications WHERE expires_at < now() - interval '1 day';
DELETE FROM password_resets WHERE expires_at < now() - interval '1 day';
```

To run it on a schedule, make it a job of the module, as
[Background work](../../../docs/extending.md#background-work) shows.

## Ports

`usecase/ports.go` declares what the use cases need:

- `UserRepository`, `SecretRepository` (one for codes, one for reset tokens), `SessionRepository` and
  `UnitOfWork`, implemented by `gateway/postgres`;
- `Mail`, implemented by `gateway/mail`, which writes the messages from embedded templates (text with an HTML
  version) and queues them on the platform mailer;
- `PasswordHasher` (`platform/password.Bcrypt`), `Tokens` (`platform/auth/jwt.Issuer`), `Randomness`
  (`platform/random.Generator`), `Clock` (`platform/clock.Real`) and `IDGenerator` (`platform/id.Generator`).

Changes to a user go through `UserRepository.Update`, which holds the user's row lock; rotations and
revocations of sessions take the same lock, so they apply one after the other per user.

`domain.User` is an entity: its state is unexported and changes only through its methods. `domain.Session` and
`domain.Secret` are not entities but value records that the use cases build and the gateways store and load,
so they keep exported fields; their rules sit in the use cases, the store's conditional statements and
`Secret.Check`. A new module models its entities on `User`, not on them.

## Public API

- `contract/`: `User` (ID, display name), `ErrNotFound` and the role names `RoleReader` and `RoleEditor`.
  Standard library only.
- `api.go`: `(*Module).API()` returns `API`, whose `User(ctx, id)` looks a user up. A consumer wraps it in its
  own `gateway/identity/`.
- `(*Module).Provision(ctx, input)`: a verified account with roles, for seed data; no route reaches it.

## Dependencies

No other module. The application gives it the mail queue and the name of the service, shown in the messages.

## Reading order

1. `domain/`: `user.go`, `email.go`, `password.go`, `secret.go`, `session.go`, then `errors.go`.
2. `usecase/ports.go`, `user.go` (`UserUsecase`, `UserView`, `TokenPair`, `Settings`), then `register.go`,
   `login.go` and `refresh.go`.
3. `gateway/postgres/`: `user_repo.go`, `session_repo.go`, then `session_rotate.go`; `gateway/mail/`.
4. `transport/rest/`: `routes.go`, `handler.go`, then one endpoint, such as `login.go`, and `errors.go`.
5. `contract/contract.go` and `api.go`: what other modules see.
6. `config.go`, then `module.go`.
