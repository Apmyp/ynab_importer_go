---
description: "Run user story in TDD loop"
argument-hint: "STORY_NUMBER (e.g., 001)"
---

# Story Command

Run a user story using Ralph loop with feature-dev and TDD workflow.

## Instructions

1. Parse the story number from arguments: `$ARGUMENTS`

2. Find the story file matching pattern `.claude/stories/${STORY_NUMBER}-*.md`

3. Read the story file and extract:
   - `id` from frontmatter
   - `title` from frontmatter
   - `branch` from frontmatter
   - Full content (description, acceptance criteria, DoD)

4. Check if branch exists, if not create it:
   ```bash
   git checkout -b <branch> || git checkout <branch>
   ```

5. Start Ralph loop with the story task. Use `/ralph-loop` command:

   ```
   /ralph-loop "<STORY_PROMPT>" --completion-promise "STORY COMPLETE" --max-iterations 50
   ```

   Where `<STORY_PROMPT>` is built from the story content below.

## Story Prompt Template

Build the prompt for ralph-loop:

```
You are working on user story: <TITLE>

Branch: <BRANCH>

<FULL_STORY_CONTENT>

## Workflow

Use /feature-dev to implement this story following TDD:

1. Start with /feature-dev to understand requirements and plan
2. Write a small failing test first
3. Write minimal code to pass the test
4. Run tests: go test ./...
5. If green, refactor if needed
6. Repeat until all acceptance criteria met

## Completion Requirements

Before completing, verify ALL of these:

1. All acceptance criteria are checked off
2. All DoD items are satisfied
3. Tests pass: go test ./...
4. Coverage > 90%: go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep total
5. Code compiles: go build ./...
6. Code formatted: gofmt -l . (should return empty)

When ALL requirements are met, output:
<promise>STORY COMPLETE</promise>

DO NOT output the promise until everything is verified!
```

## On Completion

When the loop completes (promise detected), run pre-commit checks and commit:

```bash
.claude/scripts/pre-commit-story.sh && git add -A && git commit -m "feat(<STORY_ID>): <TITLE>"
```

Report the final status to the user.
