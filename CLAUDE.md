# CLAUDE.md

The README covers the rulebook and the golden races. CONTRIBUTING.md has the rules for a change.

## Local validation

GitHub Actions CI is off to save paid minutes, so nothing checks a pull request on GitHub. Run `scripts/check.sh` and make it pass before you push. It runs what CI ran: golangci-lint v2.13.2 (built with `go run`, as CI built it), `go build`, `go vet` and `go test ./...`. The tests are fast, so there is no `--full` mode. Commit messages and pull request titles still have to follow Conventional Commits for release-please, and nothing checks them now.

The pre-push hook in `.githooks/pre-push` runs `scripts/check.sh`. Claude Code sessions enable it on start; humans enable it with `git config core.hooksPath .githooks`.
