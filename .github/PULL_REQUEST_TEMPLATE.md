## Summary

<!-- What is true once this lands, and why. Link the issue it settles, if any (Closes #N). -->

## Checklist

<!-- Tick what applies; remove what does not. AGENTS.md and CONTRIBUTING.md have the rules behind each item. -->

- [ ] The branch starts from `master` and is named `<type>/<what it does>`
- [ ] Commit messages read `<type>(<scope>): <title>` with a body of one paragraph
- [ ] A CHANGELOG entry under `## Unreleased`
- [ ] The definition of done passes: build and vet with and without `-tags=integration`, `make test`,
      `make test-integration`, `make lint`, `make generate-check` and `make openapi-check`
- [ ] New settings are in `.env.example` and the README's configuration table
- [ ] The docs that describe the change are updated: the README, a module card or `docs/extending.md`
