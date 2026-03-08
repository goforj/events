# Repository Guidelines

## Project Structure & Module Organization
- `cmd/forj`: main entrypoint; builds the `goforj` CLI (see `bin/forj` for a prebuilt binary).
- `internal/forj`: core commands (make:*, dev, render, run), project renderer, and templates; `internal/cmd` hosts the top-level command set.
- `wire`: dependency injection setup (edit `wire.go`/helpers, regenerate `wire_gen.go` via `go generate ./wire`).
- `env`: environment loading (`.env`, `.env.testing`, `.env.host`) and runtime detection; `crypt` for crypto helpers; `internal/migrations` for migration plumbing.
- `docs`: VitePress documentation; `containers/test-build` for containerized checks. Tests live next to their packages (`*_test.go`).

## Build, Test, and Development Commands
- `go build ./cmd/forj` builds the CLI; `go run ./cmd/forj --help` for a quick smoke check.
- `go test ./...` for the full suite; add `-cover` for coverage. Use temp dirs for filesystem-heavy tests.
- `go generate ./wire` after modifying DI providers/sets.
- `npm run docs:dev` / `npm run docs:build` to work on the docs site (run from repo root).
- `goforj dev` drives scaffolded project watchers defined in `.goforj.yml` (requires `wgo` installed locally).

## Coding Style & Naming Conventions
- Target Go 1.23+. Run `gofmt -w .` (or `goimports`) before pushing; keep imports grouped.
- Packages stay cohesive: CLI logic in `internal/forj`, shared utilities in `env`/`crypt`. Avoid editing generated files (e.g., `wire_gen.go`) manually.
- Use PascalCase for types, camelCase for locals, and descriptive flag/command names. Prefer structured logging via `logger.AppLogger` (zerolog) over string concatenation.

## Testing Guidelines
- Favor table-driven tests colocated with code; mirror filenames with `_test.go`.
- Load env defaults with `env.LoadEnvFileIfExists()` when behavior depends on APP_ENV; reset any global env/state per test.
- For commands that write files (renderers, generators), use temp dirs and clean up artifacts; capture stderr/stdout where helpful.

## Commit & Pull Request Guidelines
- Use conventional commits (e.g., `chore:`, `fix:`, `feat:`) as seen in history; keep changesets focused.
- PRs should include intent, linked issues, commands run, and relevant output or screenshots (CLI/log snippets). Note when Wire was regenerated or when `.goforj.yml` expectations change.
- Update docs/examples alongside behavior changes to keep scaffolds and guidance in sync.

## Security & Configuration Tips
- Never commit secrets; keep `.env`, `.env.testing`, and `.env.host` local. APP_ENV defaults to `local` unless explicitly set.
- Dev watchers strip existing `APP_*` vars—export needed values in your shell before running `goforj dev`.
- Render stress tests cache modules/build artifacts under the OS cache (`~/Library/Caches/goforj` on macOS); clear these if encountering stale state.

Absolutely. Below is an **additive extension** you can drop directly under the existing content in `AGENTS.md`. It codifies the *philosophy*, *constraints*, and *agent expectations* we’ve discussed across GoForj, without repeating what you already wrote.

You can paste this verbatim after the current sections.

## GoForj Philosophy & Design Constraints

### Intent of the Project

GoForj is a **cohesive Go application toolchain**, not just a CLI or framework. Every package, command, and scaffold should:

* Reduce decision fatigue for developers
* Encode best practices by default
* Optimize for *developer flow* over raw flexibility
* Feel predictable, composable, and safe to extend

When in doubt, favor **clarity, convention, and leverage** over abstraction for abstraction’s sake.

### Opinionated by Default, Escape Hatches Explicit

* Defaults should be sensible, production-oriented, and boring.
* Advanced customization is allowed, but should require *intentional opt-in*.
* Avoid “magic” behavior that cannot be traced through code or config.
* Prefer config-driven behavior (`.goforj.yml`, env) over hidden heuristics.

If something is surprising, it’s likely wrong.

## Performance, Allocation, and Immutability Guidelines

* Avoid unnecessary allocations in hot paths (rendering, generators, pipelines).
* Prefer reuse of backing slices/structures **when mutation is explicit and documented**.
* Immutability is a *tool*, not a religion — cloning should be deliberate and visible.
* APIs should make mutation obvious via naming (`InPlace`, `Mutate`, `Unsafe`, etc.).

Benchmark-informed decisions are encouraged; performance regressions must be justified.

## Agent Expectations (Human or AI)

When contributing changes, agents should:

* **Preserve architectural intent** — do not simplify at the cost of future leverage.
* **Avoid introducing parallel abstractions** unless a clear gap exists.
* **Prefer extending existing patterns** over inventing new ones.
* **Leave the codebase more legible than before** (comments, naming, structure).

If a change cannot be explained clearly in a PR description, it likely needs rethinking.

## API & Developer Experience Principles

* APIs should read fluently left-to-right and compose naturally.
* Naming should optimize for discoverability in IDE autocomplete.
* Avoid boolean arguments where intent is unclear; prefer options/builders.
* Public APIs should be hard to misuse and easy to do the right thing with.

Breaking changes are acceptable **only** when they materially improve long-term DX.

## Scaffolds, Templates, and Generated Code

* Generated output is part of the product — treat it as first-class code.
* Templates should favor explicitness over terseness.
* Generated code should be readable, idiomatic Go that users won’t feel compelled to rewrite.
* Any change to templates requires validating:
    * Fresh scaffold
    * Upgrade path (where applicable)
    * Docs consistency

Never generate code you wouldn’t maintain by hand.

## Documentation as a Product Surface

* Docs are not an afterthought — they define how GoForj is perceived.
* Every meaningful behavior change should be reflected in:
    * CLI help output
    * VitePress docs
    * Examples (where applicable)
* Prefer concrete examples over abstract explanations.

If behavior is undocumented, it is effectively unsupported.

## Long-Term Stability Rules

* Avoid locking GoForj into transient ecosystem trends.
* Prefer standard library + small, proven dependencies.
* Treat Wire, Kong, and watchers as infrastructure — changes here ripple widely.
* Backward compatibility matters more for scaffolds than internals.

Optimize for **years of use**, not quick wins.

## Building a GoForj Library

This section is for engineers creating a new standalone GoForj library, such as `events`, `storage`, `cache`, or `queue`.

Treat the library as a product, not as an internal helper package. The bar is:

* clear name and mental model
* small, coherent API
* strong examples and generated API docs
* explicit testing story
* clean module boundaries
* predictable release mechanics

If the library is confusing to discover, difficult to test, or expensive to depend on, the design is not done.

### Product Framing

Before writing code, lock down these decisions:

* What is the actual abstraction?
* What should the library be called?
* What should it explicitly **not** promise?
* Which features belong in the core interface versus optional capability surfaces?

Do not let implementation details define the product language. Name the library after the abstraction users work with, not the mechanism underneath.

Examples:

* `storage`, not `filesystem`, because the real abstraction is storage over object-like backends.
* `cache`, not `redis-wrapper`, because the abstraction is cache, not one driver.

### Library Shape

Default shape for a serious GoForj library:

* root module for the core contract
* per-driver or per-backend submodules
* a shared test helper module
* a centralized integration module
* an examples module
* docs generators where needed

For a driver-based library, prefer:

* `github.com/goforj/<lib>`
* `github.com/goforj/<lib>/driver/<backend><lib>`
* `github.com/goforj/<lib>/<lib>test`
* `github.com/goforj/<lib>/integration`
* `github.com/goforj/<lib>/examples`

The root module should stay thin. It should own:

* public interfaces
* errors
* manager/registry/factory APIs
* normalization/contract helpers

It should not pull in every backend SDK unless there is a strong reason.

### Dependency Boundaries

Keep heavy dependencies in backend modules, not in the root.

That is the standard pattern:

* root module: contract and orchestration
* backend module: SDK dependency and concrete implementation

This matters for:

* dependency weight
* compile times
* long-term maintenance
* user trust

If one driver needs a large dependency tree, isolate it. Do not push that cost onto every consumer.

### Naming Conventions for Drivers

Prefer backend names that do not force import aliases in normal use.

For example, this pattern is good:

* `driver/rediscache`
* `driver/localstorage`
* `driver/rclonestorage`

This is better than:

* `driver/redis`
* `driver/local`
* `driver/rclone`

because the package name is clearer at call sites and users do not need aliases for normal imports.

The package and directory name should usually match.

### Public API Standards

Keep the default API small.

Only put methods on the main interface when they are:

* broadly useful
* semantically coherent across implementations
* worth documenting and testing across all drivers

If a method is awkward to explain or impossible to support consistently, it probably does not belong in the core interface yet.

For GoForj libraries, default to:

* ergonomic first
* explicit second
* magical never

Do not require ceremony for the 95% path if the advanced path can stay explicit.

### Context Strategy

Do not force `context.Context` through every default method unless it materially improves everyday usage.

Preferred pattern:

* context-free primary interface
* `*Context` variants for advanced control

Example:

* `Get(...)`
* `GetContext(ctx, ...)`

Use `MethodContext`, not `MethodCtx`, for exported methods. This matches common Go library patterns such as `QueryContext` and `CommandContext`.

Docs should treat the context-free API as primary. Context-aware variants should be documented as the advanced path for cancellation and deadlines.

### Config Strategy

Do not build giant union config structs if the library has multiple backends.

Preferred pattern:

* typed config per backend module
* root manager/build APIs accept a driver config interface
* internal resolved config objects are adapter glue, not the user-facing DX surface

This gives:

* better autocomplete
* clearer docs
* easier per-driver evolution
* lower misuse risk

The public surface should optimize for what users should write by hand, not what is easiest to thread through internals.

### Construction Paths

If the library supports pluggable backends, support these paths:

* direct constructor per backend
* root single-backend builder
* manager for named, config-driven instances

That means users should be able to:

* construct one backend directly for simple code or tests
* construct one backend through the shared abstraction
* manage multiple named backends in apps

If the library only supports one construction style, it is probably underserving at least one real use case.

### Manager and Named Resource APIs

Named resource lookup by string is acceptable for config-driven infrastructure, but it is runtime-bound and inherently weaker than typed construction.

Keep manager APIs small and honest. Let consuming apps wrap them with generated, typed accessors if they want compile-time ergonomics.

Do not push app-specific naming schemes into the core library.

### Testing Standards

Every serious GoForj library should have three layers:

1. unit tests next to each package
2. shared contract tests
3. centralized integration tests

The centralized integration suite should be authoritative. Do not maintain duplicate, overlapping per-driver integration suites unless they validate something the centralized contract cannot.

For backend libraries, the preferred model is:

* one contract harness
* one integration module
* iterate over all real fixtures
* run the same assertions for each backend

This reduces drift and proves that “supported” means something concrete.

### Test Infrastructure

Use `testcontainers-go` where it makes the suite more realistic and stable.

Use emulators or embedded servers when they are materially simpler and still credible.

The important standard is:

* one shared integration battery
* covering all bundled drivers that can be reasonably exercised

Do not keep stale `docker-compose` infrastructure around once the centralized test strategy has moved on.

### Fakes and Developer Testing Ergonomics

If the library benefits from fake backends for tests, provide them.

Do not copy framework magic directly. Instead, provide Go-native ergonomics:

* an in-memory backend where appropriate
* test helper constructors in `<lib>test`
* fake manager helpers if the library uses named resources

The test story is part of the product surface.

If developers have to assemble their own fake backend wiring from scratch every time, the library is underserving real usage.

### Coverage Expectations

Integration coverage and unit coverage solve different problems.

Use both.

If the repo is multi-module:

* collect coverage per module
* merge coverage into one artifact for reporting
* include the integration module in the merged coverage path

Do not assume integration tests alone will produce blanket branch coverage. They usually will not. Use unit tests to close the branch and error-path gaps.

When coverage is low, focus first on:

* root contract packages
* bundled backends
* shared contract helpers

Treat test helper plumbing as real code, but do not confuse it with the highest-value coverage target.

### Documentation Requirements

Documentation is part of the API.

Every public, user-facing API should have:

* a clear doc comment
* a runnable example where it materially helps users

If the library has doc-comment-driven docs generation, keep it healthy. Do not let the README and generated API docs drift apart.

Preferred pattern:

* short product README at the top
* generated API index lower down
* compile-checked generated examples

The README should explain:

* what the library is
* what drivers/backends it supports
* how to install it
* how to do the common things first

Do not fill the README with internal migration history or internal architecture notes.

### Example Generation

If the library uses generated examples from doc comments:

* treat the doc comments as the source of truth
* keep the generated examples runnable
* keep the README API section generated
* add compile checks for generated examples in CI

Generated examples should look like real usage, not internal scaffolding artifacts.

Avoid:

* redundant aliases
* fake block scopes
* noisy throwaway lines in rendered README examples

### Capability Matrices

If the library has multiple drivers or backends, include a matrix near the top of the README.

Use one table for:

* drivers/backends and high-level notes

Optionally add a second table for:

* capabilities, with simple support markers

Keep the top table product-focused. Put lower-level capability details in a separate table or dedicated support document.

### CI Expectations

For multi-module libraries, CI should:

* run unit tests
* run centralized integration tests
* compile generated examples
* merge coverage across modules

Use one CI workflow with parallel jobs when possible rather than multiple overlapping pipelines.

Use `actions/setup-go` caching with `cache-dependency-path` covering all module `go.sum` files.

Do not add docs generation as a CI mutation step if contributors are expected to run generators locally. CI should verify generated outputs, not rewrite them silently.

### Release and Tagging

If the repo uses multiple Go modules, provide a tagging utility that can tag all modules correctly.

Do not expect engineers to remember tag prefixes by hand.

For example:

* root: `v0.1.0`
* driver module: `driver/localstorage/v0.1.0`
* integration module: `integration/v0.1.0`

Automate this with a repo script and make target.

### Design Discipline

When shaping a new library, ask these questions repeatedly:

* Is this API honest about what it guarantees?
* Is the common path easy?
* Are heavy dependencies isolated?
* Are docs and examples reflecting the real intended usage?
* Is the integration suite proving the core contract across the bundled drivers?

If the answer to any of those is no, the library is not ready yet.
