# Contributing

Thank you for your help. There are two types of change.

## Rulebook files

Dansk Orienterings-Forbund (DO-F) sets the rules. This repository only
implements them. Each file in the `rulebooks` folder is one version of the
DO-F rules. You do not need to know Go.

- **A new version of the rules** goes in a new rulebook file. Do not change
  the old file. The README tells you how in "When the rules change".
- **An error in a rulebook file** is a value that does not agree with that
  version of the DO-F rules and so gives a wrong calculation. Only this
  type of error is a reason to change an existing file.

To correct an error:

1. Read the section "The rulebook" in the [README](README.md).
2. Change the value in the YAML file.
3. In the pull request, give the section of the DO-F rules for the value.

Each pull request runs the tests. A failed test shows that the change gives
a different result for a known race. Then you must show which DO-F rule
gives the new result.

## New races

A real race with an official result makes the tests stronger. The README
tells you how to add one in "Add a race". Use generic runner names only.

## Code changes

1. Run `go test ./...` before you open a pull request.
2. Run `golangci-lint run` with the version in `.github/workflows/ci.yml`.
3. Add a test for each change in the calculation.
4. Calculate the expected numbers in a test by hand from the rules, not from the code.
5. Write comments that tell why the code does a thing.

Use only the Go standard library and `gopkg.in/yaml.v3` as dependencies.

## Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/), for
example `fix: share points on equal times`. The release tool uses them to
make the changelog.
