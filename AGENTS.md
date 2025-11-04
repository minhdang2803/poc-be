# Repository Guidelines

## Project Structure & Module Organization
The repository is laid out for a Go service. Place executable entrypoints under `cmd/mmispoc/main.go` to coordinate application modules. Shared, non-exported packages belong in `internal/`, grouped by domain such as `internal/service` and `internal/repository`. Reusable libraries that might be consumed externally should go in `pkg/`. Store configuration fixtures under `configs/` (for example, `configs/local.yaml`) and check in sample assets under `assets/` or `testdata/` as needed. Keep scripts that wire local tooling in `scripts/`.

## Build, Test, and Development Commands
Run `go mod tidy` after adding dependencies to keep `go.mod` clean. Use `go build ./cmd/mmispoc` to verify the binary compiles, and `go run ./cmd/mmispoc` for a quick local boot with default configs. Execute `go test ./...` for the full unit suite; add `-race` when debugging concurrency issues. Static analysis via `go vet ./...` should succeed before opening a PR. Optional: `golangci-lint run` if you have the aggregator installed.

## Coding Style & Naming Conventions
Rely on `gofmt` (or `go fmt ./...`) and `goimports` before committing; Go’s canonical formatting (tabs for indentation, single import blocks) is required. Types, interfaces, and functions use PascalCase; package-level variables are camelCase unless exported. Keep packages cohesive and avoid stutter (for example, rename `service.UserService` to `service.User`). Prefer constructor functions named `NewX`, and guard errors with `errors.Is` or `errors.As`.

## Testing Guidelines
Unit tests live next to their source files with the `_test.go` suffix and exported `TestXxx` functions. Favor table-driven tests and clear arrange/act/assert sections. Use `go test -cover ./...` to confirm coverage and target at least 75% on new packages. Integration tests that hit external systems should be tagged and skipped by default using `//go:build integration`, then invoked explicitly with `go test -tags=integration ./...`. Drop fixtures under `testdata/` so they’re picked up automatically by the Go toolchain.

## Commit & Pull Request Guidelines
Adopt Conventional Commits to keep history searchable (for example, `feat: add customer lookup handler` or `fix: guard nil cache client`). Each pull request should include a short summary, any linked issue ID (such as `Closes #123`), and validation notes describing the commands you ran. Attach screenshots or JSON samples when you modify user-facing schema. Rebase onto `main` before requesting review, and ensure CI passes cleanly.
