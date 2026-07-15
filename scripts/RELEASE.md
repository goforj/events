# Releasing Events Modules

This repository contains public Go modules and repo-only support modules. They
have different release boundaries.

## Published Modules

`scripts/published-modules.txt` is the dependency-ordered source of truth for
the modules that receive tags. The current order is:

1. `github.com/goforj/events/eventscore`
2. `github.com/goforj/events`
3. `github.com/goforj/events/eventstest`
4. `github.com/goforj/events/eventsfake`
5. `github.com/goforj/events/driver/gcppubsubevents`
6. `github.com/goforj/events/driver/kafkaevents`
7. `github.com/goforj/events/driver/natsevents`
8. `github.com/goforj/events/driver/natsjetstreamevents`
9. `github.com/goforj/events/driver/redisevents`
10. `github.com/goforj/events/driver/snsevents`

Every published module must use the coordinated release version for sibling
requirements and must not contain a `replace` directive. `go.work` supplies
version-specific local sibling resolution while developing and in pre-release
CI. The workspace replacements are tooling boundaries; they are never part of
a published module.

The `eventscore` API remains compatible with `v0.1.3`, but the quality-pass
release intentionally coordinates every public module at `v0.2.0`. This keeps
one repository commit associated with one public sibling version instead of
mixing old and new tags across the module graph.

## Support Modules

These modules are workspace-only verification and documentation surfaces:

- `github.com/goforj/events/docs`
- `github.com/goforj/events/examples`
- `github.com/goforj/events/integration`

They do not receive release tags. They may use local replacements because they
are not part of the downstream dependency contract.

## Before Tagging

Run the static module-boundary check and the workspace quality gates:

```sh
bash scripts/check-published-modules.sh
GO_TEST_FLAGS='-race -count=1' sh scripts/test-all-modules.sh

for dir in . eventscore eventstest eventsfake docs examples \
  driver/gcppubsubevents driver/kafkaevents driver/natsevents \
  driver/natsjetstreamevents driver/redisevents driver/snsevents integration
do
  (cd "$dir" && go vet ./...)
done

bash scripts/check-tidy-modules.sh

sh scripts/check-docs.sh
git diff --exit-code
```

Integration tests need their external services and are run separately:

```sh
GO_TEST_FLAGS='-race -count=1' sh scripts/test-integration.sh
```

Preview the exact tag set from a clean release commit:

```sh
scripts/tag-all-modules.sh v0.2.0 --dry-run
```

The preview must contain ten tags and must not contain `docs/`, `examples/`, or
`integration/` tags.

## Tagging

Create every public tag from the same commit and push them as one atomic remote
update:

```sh
scripts/tag-all-modules.sh v0.2.0 --push
```

The script rejects a version that differs from the published sibling
requirements and runs the module-boundary guard before creating tags. If the
remote rejects any ref, `git push --atomic` leaves the entire remote tag set
unchanged. A retry with `--skip-existing` reuses tags only after verifying that
their peeled commit is the current release commit.

## Public-Proxy Validation

Do not add pre-tag `GOWORK=off` consumer checks for the planned sibling version.
Before the tags exist, those checks can only fail because `v0.2.0` is not
available to the public module proxy yet; that does not test the release commit.
The tidy script uses temporary module files with local sibling bindings so it
can still verify all committed module and checksum content before tagging.

After all tags are pushed and visible through the proxy, validate every exact
module version from fresh temporary consumers:

```sh
scripts/check-public-release.sh v0.2.0
```

The script explicitly downloads `module@v0.2.0`, verifies that exact version
was selected from a fresh module cache, and tests packages from the downloaded
module. Its default public proxy has no direct-VCS fallback; set
`EVENTS_RELEASE_GOPROXY` to another proxy-only endpoint for a private mirror.
Running `go mod download` or `go test` in the repository alone is insufficient
here because it can validate the local checkout without fetching the module's
own release tag.

If resolution fails after the tags are visible, first confirm that every tag
points to the same release commit and that the public module's `go.mod` has no
local replacement.
