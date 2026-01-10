# Go TDD User Story Development

This project uses TDD-first workflow for Go development with user stories.

## Workflow

1. User selects a story by number (e.g., `/story 001`)
2. System creates branch `feature/001-<short-name>`
3. Ralph loop starts with feature-dev inside
4. TDD cycle: small test -> small code -> green -> refactor -> repeat
5. On completion: gofmt, tests, coverage check, commit to branch

## TDD Rules (STRICT)

1. **Write test FIRST** - Always write a failing test before implementation
2. **Minimal code** - Write only enough code to pass the current test
3. **Small steps** - Each test should test ONE behavior
4. **Run tests frequently** - After every small change
5. **Refactor only when green** - Never refactor with failing tests
6. **Coverage > 90%** - Required before story completion

## Go Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with coverage percentage
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep total

# Run specific test
go test -run TestName ./...

# Verbose output
go test -v ./...
```

## Go Format & Build

```bash
# Format code
gofmt -w .

# Check if code compiles
go build ./...

# Vet code
go vet ./...
```

## Story Completion Checklist

Before outputting `<promise>STORY COMPLETE</promise>`:

- [ ] All acceptance criteria met
- [ ] All DoD items satisfied
- [ ] Tests pass: `go test ./...`
- [ ] Coverage > 90%: check with `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep total`
- [ ] Code compiles: `go build ./...`
- [ ] Code formatted: `gofmt -l .` returns empty

## Story File Format

Stories are in `.claude/stories/NNN-short-name.md`:

```markdown
---
id: "001"
title: "Short title"
branch: "feature/001-short-name"
---

# Title

## Description
What needs to be done.

## Acceptance Criteria
- [ ] Criterion 1
- [ ] Criterion 2

## Definition of Done
- [ ] Tests pass
- [ ] Coverage > 90%
- [ ] Code compiles
- [ ] Code formatted
```
