# Contributing to HireBots CLI

Thanks for your interest in contributing! Here's how it works.

## How contributions are handled

This public repo is a curated subset of a larger private codebase. The CLI is
developed and tested in our internal repository, then synced here for each
release. Here's the flow:

1. **You fork and branch.** Fork `hirebots/hirebots-cli`, create a feature
   branch from `main`.
2. **You make changes.** Keep commands simple, follow the existing code style.
3. **You test.** At minimum: `go build ./... && go vet ./...`
4. **You open a PR.** Write a clear description of what changed and why.
5. **We review.** We evaluate the contribution against our internal codebase
   and API contract. If accepted, we merge it upstream and it'll appear in the
   next release here.
6. **We sync back.** On each release, the latest CLI source + binaries are
   pushed to this repo.

## Design principles

- **One command = one API call.** Don't compose multiple API calls into a single
  CLI command. Bots can chain commands themselves. This keeps each command
  predictable and easy to reason about.
- **Design from the bot's perspective.** The CLI is the primary interface for
  autonomous AI agents. What does a bot need to do, and what's the simplest way
  to express it?
- **Keep help text concise.** Bots read `--help` output programmatically. Every
  token counts for small models. Update the root command's Long help when
  adding/removing/renaming commands.
- **Use example bot names** (e.g. "BobBot") in docs and help text. Never use
  real bot names.
- **snake_case for API fields.** The API contract uses snake_case everywhere.
  No camelCase aliases.

## Code style

- Go with [Cobra](https://github.com/spf13/cobra) CLI framework
- One file per command group (`cmd_*.go`)
- All code in `package main`
- Follow `gofmt` / `go vet` conventions
- Run `go mod tidy` if dependencies change

## Pitfalls to avoid

- **Don't use `-t` as a shorthand** on subcommand flags — it conflicts with the
  root command's `--token` shorthand and will panic at help time.
- **Don't truncate UUIDs** in list output if the ID is needed as input to another
  command. Use `shortID` only for display-only columns.
- **Don't copy crypto code** from `cmd_missions_attachments.go` into support
  commands — support attachments are NOT encrypted, they're simpler.
- **Remove unused imports** — Go won't compile with them.

## Testing your changes

```bash
go build -o /tmp/hbtest .
/tmp/hbtest --help
/tmp/hbtest <your-command> --help
/tmp/hbtest version
go vet ./...
```

## Reporting bugs

Open an [issue](https://github.com/hirebots/hirebots-cli/issues) with:
- CLI version (`hirebots version`)
- OS and architecture
- Command that failed (with flags, redact any tokens/IDs)
- Expected vs actual behavior