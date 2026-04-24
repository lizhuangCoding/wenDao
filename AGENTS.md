# Repository Guidelines

## Project Structure & Module Organization

This repository contains a Go backend and a Vite React frontend.

- `backend/`: Go module `wenDao`; server entrypoint is `cmd/server/main.go`.
- `backend/internal/`: handlers, middleware, models, repositories, services, and shared packages under `internal/pkg`.
- `backend/config/`: YAML config plus optional `config/.env` loaded at startup.
- `backend/migrations/`, `backend/uploads/`, `backend/log/`: database, uploaded files, and local runtime output.
- `frontend/src/`: API clients, components, hooks, pages/views, router, store, styles, types, and utilities.
- `docs/`: design docs, specs, and implementation plans.

## Build, Test, and Development Commands

From `backend/`:

- `go run ./cmd/server`: start the API on port `8089`.
- `go test ./...`: run all backend tests.
- `go build ./cmd/server`: compile the backend server binary.
- `go fmt ./...`: format Go code before committing.

From `frontend/`:

- `npm ci`: install dependencies from `package-lock.json`.
- `npm run dev`: start Vite on `localhost:3000`; proxies `/api` and `/uploads` to `8089`.
- `npm run build`: type-check with `tsc` and build production assets.
- `npm run lint`: run ESLint with zero-warning enforcement.
- `npm run preview`: preview the production build.

## Coding Style & Naming Conventions

Use standard Go formatting and package names: lowercase package directories, exported names in `PascalCase`, private names in `camelCase`. Keep handlers thin; place business logic in `internal/service` and persistence in `internal/repository`.

Frontend code uses TypeScript, React function components, Tailwind utilities, and the `@` aliases in `vite.config.ts`. Use `PascalCase` for components, `camelCase` for utilities, and `useXxx` for hooks.

## Testing Guidelines

Backend tests use Go’s `testing` package and live beside covered code as `*_test.go`. Add focused tests for service, repository, handler, and config behavior, then run `go test ./...`.

No frontend test runner is configured. For frontend changes, run `npm run build` and `npm run lint`; include manual verification notes for UI behavior.

## Commit & Pull Request Guidelines

Recent history uses Conventional Commit-style subjects, for example `feat: add chat handler with CRUD API`, `fix: add /api prefix to chat API paths`, and `chore: add .worktrees/ to gitignore`. Keep commits scoped and imperative.

Pull requests should include a short summary, test results, linked issue or design doc when applicable, and screenshots for visible frontend changes. Call out configuration, migration, or data-shape changes.

## Security & Configuration Tips

Do not commit new secrets. Override local credentials with `backend/config/.env` or environment variables such as `DB_HOST`, `JWT_SECRET`, `DOUBAO_API_KEY`, `GITHUB_CLIENT_ID`, and `GITHUB_CLIENT_SECRET`. The backend rejects the placeholder JWT secret at startup.
