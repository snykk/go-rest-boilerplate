# Security policy

go-rest-boilerplate is copied into services that face the internet, so a weakness in it spreads to every
project built from it. Reports of vulnerabilities are welcome.

## Reporting a vulnerability

Report privately through GitHub's private vulnerability reporting: open the repository's **Security** tab
and choose **Report a vulnerability**. Please do not open a public issue.

A useful report includes:

- the affected version or commit,
- what happens and what an attacker gains,
- steps to reproduce, such as the request and the response.

## Scope

In scope are bugs in this repository that let a request or a configuration:

- read or change data it should not, or act as another caller, such as getting past `auth.Authenticate` or
  `auth.Require`;
- reach the operator endpoints under `/admin` without `ADMIN_TOKEN`;
- carry a secret or a submitted value into a response, a problem document or a log record;
- forge the client IP or another value the server trusts;
- crash the process, or exhaust its memory or connections beyond the configured limits.

Out of scope are vulnerabilities in dependencies that are already public, which the weekly `govulncheck`
scan tracks and which belong upstream; weaknesses of the deployment around the service, such as TLS or the
database's network; and code that a project changed after copying the boilerplate.

## Supported versions

Before 1.0, fixes go to the latest release only. A fix ships as a new release, and a published tag never
moves.
