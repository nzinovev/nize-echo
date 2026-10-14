# nize-echo

Bilingual (RU/EN) personal site: a blog organized by contexts and tags, a CV, and later paid courses. One Go binary serves server-rendered public pages, the admin JSON API and the embedded React admin; Postgres holds all content. Content lives in the database, code lives in git: publishing never needs a deploy.

## Sources of truth

- `design/Personal Site — Concept.md`: pages, behavior, content model, architecture, acceptance criteria AC-1 to AC-22.
- `design/nize-echo — v1 Backlog.md`: tasks (IDs like F-01, C-02), their dependencies and *Done when* checks, and the shared definition of done.
- `design/system/`: tokens, `nz-` CSS classes and components.

When they disagree, the concept wins for behavior and the design system wins for looks.

## Commands

| Command | What it does |
| --- | --- |
| `make help` | List make targets (default goal) |
| `make build` | Build the server binary into `bin/site` |
| `make run` | Build and run the server; migrations apply on start (`DATABASE_URL` defaults to the compose Postgres) |
| `docker compose up -d` | Start the dev Postgres 16 (`compose.yaml`; host port `POSTGRES_PORT`, default 5432) |
| `make dev` | Run `dev-go` and `dev-admin` together; the site is at `:8080`, the admin at `/admin/` |
| `make dev-go` | Run the Go server with live reload (air, `.air.toml`) |
| `make dev-admin` | Run the Vite dev server on `:5173` (installs `admin/node_modules` if needed) |
| `make test` | Run all Go tests with the race detector; DB tests skip if the compose Postgres is down (`TEST_DATABASE_REQUIRED=1` makes them fail instead) |
| `make lint` | Run golangci-lint (config in `.golangci.yml`) |
| `make fmt` | Format code with gofumpt and goimports |
| `make generate` | Generate Go code from `db/queries` into `internal/store/sqlc` (config in `sqlc.yaml`) |
| `make image` | Build the Docker image `nize-echo:dev` from `deploy/Dockerfile` (the same build CI runs) |
| `make tidy` | Tidy `go.mod` and `go.sum` |
| `make tools` | Install pinned dev tools (golangci-lint, air, sqlc) into `bin/` |
| `make clean` | Remove `bin/` |

Before finishing a task: `make build lint test` must pass. CI runs the same targets, with `TEST_DATABASE_REQUIRED=1`, plus the admin build and `make image`'s Docker build.

`.agentum.yaml` lists the checks the agent orchestrator runs on its own after an agent finishes; it mirrors CI. When a task adds, removes or changes a check (a new CI job, make target, linter step or test suite), update `.agentum.yaml` in the same change.

## Layout

| Path | What goes there |
| --- | --- |
| `cmd/site` | `main`: config, wiring, server start |
| `internal/http` | Router, middleware, public handlers, `/api/admin` handlers |
| `internal/render` | Markdown → HTML (goldmark, Chroma), shared by site and preview |
| `internal/content` | Domain logic: posts, contexts, tags, pages, settings; no HTTP |
| `internal/media` | `Storage` interface, upload validation, WebP variants |
| `internal/auth` | Passwords, sessions, rate limiting, CSRF |
| `internal/store` | pgx pool (`Open`), migrations (`Migrate`), sqlc-generated queries in `internal/store/sqlc` |
| `internal/store/storetest` | Test helper: a fresh Postgres database per test |
| `db/migrations` | goose SQL migrations `NNNNN_name.sql`, embedded by `db/embed.go` and applied on start |
| `db/queries` | sqlc query files; generated code in `internal/store/sqlc` is committed |
| `db/SCHEMA.md` | Maps each concept entity to its tables; post status and schema conventions |
| `web` | Package `web`: embeds `templates` and `static` into the binary |
| `web/templates` | `html/template` layouts and partials for public pages; `styleguide.html` is the design system test page at `/styleguide` |
| `web/static` | CSS (`tokens.css`, `bundle.css` copied from `design/system`), `fonts.css` and self-hosted woff2 fonts in `fonts/`, images |
| `admin/` | React + Vite + TypeScript admin app, embedded with `embed.FS`; `npm --prefix admin run build` type-checks and builds |
| `compose.yaml` | Local dev services only (Postgres); production Compose goes in `deploy/` |
| `deploy/` | Dockerfile (built from the repo root), Compose, Caddy, backup and restore docs |
| `.github/workflows/` | CI (`ci.yml`): admin build, Go build, lint, tests against a Postgres service, Docker image build; all jobs must pass to merge |
| `.agentum.yaml` | Checks the agent orchestrator runs after each agent (build, lint, test, admin build, image) |
| `bin/` | Build output and local tools (gitignored) |

## Conventions

- Stack: standard library `net/http` routing, `html/template`, Postgres 16 with pgx and sqlc, goose migrations. Config comes from environment variables.
- Every env variable is read in `cmd/site/config.go` (with its default) and nowhere else; `DATABASE_URL` (required), `SITE_ADDR`, `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`, `ADMIN_DEV_URL` so far; `.env.example` lists the dev values.
- `internal/http` names its package `http`; import it as `sitehttp`. Middleware lives in that package; response-writer wrappers must implement `Unwrap()` so `http.ResponseController` keeps working.
- Local dev: the Go server proxies `/admin/` to Vite when `ADMIN_DEV_URL` is set (`make dev` sets it), so the admin and its API share one origin and HMR runs through the proxy. The proxied pages drop the CSP header because Vite injects an inline script; nothing else does.
- Logging with `log/slog`. Wrap errors with `fmt.Errorf("...: %w", err)`.
- No new dependency without a one-line reason in the PR description.
- New logic has unit tests; database code is tested against a real Postgres, not mocks: call `storetest.New(t)` for an empty database (run `store.Migrate` on it for the schema). It reads `TEST_DATABASE_URL`, which `make test` points at the compose Postgres.
- Migrations are forward-only in practice: never edit one that has landed on `main`; add a new file. Every migration runs under a Postgres advisory lock at start-up.
- Schema: follow the conventions in `db/SCHEMA.md` (identity ids, `lang` enum with per-language child tables keyed `(parent_id, lang)`, slug `CHECK`, deferred unique positions). A schema change updates `db/SCHEMA.md` and the table list in `internal/store/schema_test.go`; run `make generate` after changing migrations or queries.
- Node for the admin is pinned in `admin/.nvmrc` (CI and the Dockerfile's `node` stage use the same major).
- Go stays at the version in `go.mod` (1.25); pick dependency and Makefile tool versions whose `go` directive does not exceed it (CI installs them with `GOTOOLCHAIN=local`).
- Public pages need no JavaScript to read and work in both RU and EN. UI strings come from translation files, not the database.
- Don't build anything the concept lists as Later or Not planned beyond what the data model needs.
- Update this file when a task adds a convention, command or directory.

## UI

- Read `design/system/README.md` before any UI work; component rules are in `design/system/components/`.
- Every page or admin screen has a mockup in `design/mockups/` (index in `design/README.md`). Match its structure, classes and states; list any deviation in the PR.
- Style only with tokens and `nz-` classes from `design/system/`. New tokens or classes go into `design/system/` first, then are copied to `web/static`: `tokens.css` as is, `bundle.css` without its first line (the Google Fonts `@import`; `fonts.css` replaces it). `TestStaticMatchesDesignSystem` fails on drift when `design/` is present.
- Fonts are self-hosted (the CSP allows only `'self'`): Spectral, Onest and IBM Plex Mono, latin and cyrillic subsets, declared in `web/static/fonts.css`.
- Static files are served under `/static/` by plain name (`Cache-Control: no-cache`) and by content-hashed name (`bundle.3f9a0c1d2e4b.css`, cached a year, `immutable`). Hashes are computed from the embedded files at start-up; relative `url()`s in CSS are rewritten to hashed names. Templates link files with `{{asset "bundle.css"}}`, never a literal `/static/` path.
