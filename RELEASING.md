# Releasing go-rest-boilerplate

Every change reaches `master` through a release branch and a pull request that the owner of the repository
reviews and merges on GitHub. Nothing is pushed to `master` directly, and a published tag is never moved: a
release that needs a fix is followed by the next version.

## Versions

Versions follow [Semantic Versioning](https://semver.org/) with a `v` prefix, and the version is the tag:
there is no version file. Binaries get it at build time (`VERSION` in `deployment/Dockerfile`), which
`internal/platform/buildinfo` reports. Before v1.0.0:

- a minor version (`v0.4.0`) adds capabilities, or changes the layout or the rules that a project built on
  the boilerplate follows;
- a patch version (`v0.3.1`) only fixes.

## Branches

- A change is made on a topic branch named after its commit type and what it does, such as
  `feat/platform-mailer`, `fix/client-ip-behind-proxies` or `docs/releasing`. Types: `feat`, `fix`,
  `refactor`, `test`, `docs`, `chore`, `ci`.
- A topic branch starts from `origin/master`. Never start a branch at another branch's tip.
- A branch that needs another branch that is not released yet takes it with a merge commit,
  `git merge --no-ff <dependency>` ("Merge branch 'feat/x' into feat/y"). Merge only direct dependencies.
- Commit messages follow AGENTS.md: `<type>(<scope>): <title>` and a body of one paragraph.
- A topic branch records what it changes in CHANGELOG.md, under `## Unreleased`: one entry per change for
  someone using the boilerplate, under Added, Changed, Removed or Fixed, without commit links, which a topic
  branch cannot know for sure until it is merged. When two topics add entries under the same heading, the
  merge into the release branch keeps both.

## A release, step by step

1. Create the release branch from `origin/master` and merge every finished topic branch into it with
   `--no-ff`, dependencies first. Git's default message is the right one:

   ```sh
   git fetch origin
   git switch -c release/vX.Y.Z origin/master
   git merge --no-ff feat/x    # "Merge branch 'feat/x' into release/vX.Y.Z"
   ```

2. Close the release branch with these commits, in this order:
   - the README, when the release changes what it describes, and always its Versions table;
   - `docs: record vX.Y.Z in the changelog`. It adds a section `## [vX.Y.Z] - YYYY-MM-DD` under
     `## Unreleased` in CHANGELOG.md and moves into it the entries the topic branches wrote under Unreleased,
     regrouped and reworded as one release, each linked to the commits that made it; Unreleased is left
     empty. It also adds the footer link
     `[vX.Y.Z]: https://github.com/snykk/go-rest-boilerplate/compare/vW...vX.Y.Z` from the previous version's
     tag `vW`. The section opens with one paragraph that sums up the release, which also becomes the tag's
     message, then follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/):

     ```text
     ## [vX.Y.Z] - YYYY-MM-DD

     One paragraph: what the release brings.

     ### Added
     - **What is new**: one line a reader can act on, then the detail. ([abc1234](https://github.com/snykk/go-rest-boilerplate/commit/<full hash>))

     ### Changed
     ### Removed
     ### Fixed
     ```

     Use only the headings the release needs, in Keep a Changelog's order. Write each entry for someone using
     the boilerplate, not as a list of commits, and end it with a link to the commits that made it, with the
     whole URL: GitHub does not turn a bare hash into a link inside a Markdown file. The topic branches are
     merged with merge commits, so the hashes stay valid on `master`.

3. Check the release branch. All of these must pass (AGENTS.md, Definition of done):

   ```sh
   go build ./... && go vet ./...
   go build -tags=integration ./... && go vet -tags=integration ./...
   make test test-integration lint generate-check openapi-check
   ```

   `bash .github/scripts/release.sh tag-message vX.Y.Z "<title>"` previews the message of the tag, and
   `bash .github/scripts/release.sh notes vX.Y.Z` the notes of the GitHub Release once the tag exists.

4. Push the topic branches and the release branch, and wait until CI passes on the release branch:

   ```sh
   git push origin feat/x release/vX.Y.Z
   gh run list --branch release/vX.Y.Z
   gh run watch <run id> --exit-status
   ```

5. Open a pull request from the release branch into `master`, titled `go-rest-boilerplate <what it brings>
   release`, such as `go-rest-boilerplate modular core release`:

   ```sh
   gh pr create --base master --head release/vX.Y.Z \
     --title "go-rest-boilerplate <what it brings> release" --body-file pr.md
   ```

   Its description says what the release brings, lists the topic branches it gathers, reports the checks of
   step 3 with the CI run of step 4, and names what is still open.

6. The owner reviews the pull request and merges it on GitHub with **Create a merge commit**, so that `master`
   keeps the release branch and its topic merges. The merge commit is titled after the pull request and its
   number, such as `go-rest-boilerplate modular core release (#1)`, with no body.

   The merge runs the `release` workflow (`.github/workflows/release.yml`). It puts the annotated tag `vX.Y.Z`
   on the merge commit, made of the pull request's title, a blank line and the paragraph that opens the
   version's changelog section, with `github-actions[bot]` as the tagger. It then publishes the GitHub
   Release, titled with the tag, whose notes are the changelog section followed by a link to the changes
   since the previous tag.

If the workflow fails after a merge, start it again once the cause is fixed: a run that finds its tag already
on the merge commit goes on to the release. To publish a tag made before the workflow existed, or to replace
a release's notes with its changelog section on `master`, run the workflow by hand:

```sh
gh workflow run release.yml -f tag=vX.Y.Z
```

## Repository settings

- Pull requests: merge commits are the way to merge a release; the default merge commit title is the pull
  request's title (GitHub appends ` (#N)`), with an empty message.
- `master`: protect it, requiring a pull request and green CI.
