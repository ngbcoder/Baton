# Contributing

## Branches
- `main` is the default branch.
- Feature branches: `feat/<short-name>`, fixes: `fix/<short-name>`.

## Pull requests
- Keep PRs focused and small.
- Run tests before submitting:
  - Dart: `cd protocol && dart analyze && dart test`
  - Go: `cd desktop && go vet ./... && go test ./...`
  - Flutter: `cd android && flutter analyze && flutter test`
- Update `todo.md` checkboxes when completing tasks.

## Commit messages
Use conventional style: `feat:`, `fix:`, `docs:`, `chore:`, `test:`.