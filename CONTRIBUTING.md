# Contributing

go-rest-boilerplate is a starting point that people copy, so a change is weighed by what it teaches as much as
by what it does. Questions, issues and pull requests are welcome.

## Before you start

- For a bug, open an issue with the **Bug report** form.
- For a new capability, or a change to the layout or the rules, open a **Feature request** first, so that we
  agree on where it belongs before you write it. The core stays minimal: [docs/extending.md](docs/extending.md)
  places most capabilities in the project that needs them.
- For a security problem, never open an issue: report it privately as [SECURITY.md](SECURITY.md) describes.

## Making a change

1. Read [AGENTS.md](AGENTS.md). Its architecture rules, conventions and definition of done apply to people
   and coding agents alike.
2. Fork the repository and create a topic branch from `master`, named after its commit type and what it does,
   such as `fix/client-ip-behind-proxies`. Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`.
3. Write commit messages as `<type>(<scope>): <title>`, a blank line, and a body of one paragraph that says
   what the commit changes and why, wrapped at 72 columns.
4. Record what your change does under `## Unreleased` in [CHANGELOG.md](CHANGELOG.md): one entry per change,
   written for someone using the boilerplate, without commit links.
5. Run the checks of the definition of done. The integration tests need Docker; AGENTS.md has a note for
   Windows.

   ```sh
   go build ./... && go vet ./...
   go build -tags=integration ./... && go vet -tags=integration ./...
   make test test-integration lint generate-check openapi-check
   ```

6. Open a pull request against `master`. Its template lists what a review looks at.

## How a change reaches master

`master` receives releases only. The maintainer gathers accepted topic branches in a release branch, moves
their changelog entries into the version's section with links to their commits, and merges the release
through a pull request of its own ([RELEASING.md](RELEASING.md)). Your commits are merged as they are, so your
pull request shows as merged once its release reaches `master`.

## AI-assisted contributions

You may use AI tools, and AGENTS.md is written for coding agents as much as for people. You remain
responsible for everything you submit: you must understand the change, be able to explain it in review, and
be sure it contains nothing you are not allowed to contribute. Mention substantial AI assistance in the pull
request description; commit messages carry no attribution lines.

## License of contributions

The project is licensed under the [MIT License](LICENSE). By contributing, you agree that your contributions
are licensed under it too.

## Conduct

Everyone taking part follows the [Code of Conduct](CODE_OF_CONDUCT.md).
