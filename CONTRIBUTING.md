# Contributing to the Rumpty CLI

Thanks for helping. This guide covers how to build, test and send a change.

## Community

Have a question or want to talk an idea through before writing code? Join us on [Discord](https://discord.gg/BnFdQrMBhC).

## Before you start

- **Bugs and small fixes:** open a pull request directly, or an issue first if you are unsure.
- **New commands or behaviour changes:** comment on the issue (or open one) so we can agree on the shape before you write code.
- Issues labelled [`good first issue`](https://github.com/cloverworks-hq/rumpty-cli/labels/good%20first%20issue) and [`help wanted`](https://github.com/cloverworks-hq/rumpty-cli/labels/help%20wanted) are ready to pick up. Comment that you are taking one so work isn't duplicated.
- **Security problems:** do not open a public issue. Follow [SECURITY.md](SECURITY.md).

## Set up

You need Go 1.26 or newer and `git`. `rumpty ssh`, `exec` and `copy` also use the `ssh`, `scp` and `rsync` on your machine.

```sh
git clone https://github.com/cloverworks-hq/rumpty-cli.git
cd rumpty-cli
go build -o rumpty .
./rumpty --version
```

## Check your change

Run what CI runs before you push:

```sh
make verify-imports   # gofmt and import order
make lint             # golangci-lint (installs the pinned version into hack/tools)
make test             # go test -race ./...
go build ./...
```

`make fix-lint` and `make imports` fix most formatting and lint issues for you.

Add or update tests for what you change. API calls are tested against mocks in `internal/api/mocks` (regenerate with `go generate ./...` after changing `internal/api/client_api.go`), and HTTP behaviour with `httptest` servers.

## Try it without touching your own login

The CLI keeps its session in your OS keychain and settings in `~/.config/rumpty`. To experiment without affecting your real login, use a throwaway config directory and keep secrets out of the keychain:

```sh
export XDG_CONFIG_HOME="$(mktemp -d)"
export RUMPTY_NO_KEYCHAIN=1
export RUMPTY_NO_UPDATE_NOTIFIER=1
./rumpty login
```

`RUMPTY_API_URL` points the CLI at another API. Plain `http://` is only accepted for `localhost`.

## Code layout

| Path | What lives there |
|---|---|
| `cmd/` | Cobra commands: flags, arguments and help text |
| `internal/api/` | HTTP client for the Rumpty API and its request/response types |
| `internal/auth/` | Login and logout |
| `internal/credentials/` | Keychain and private-file storage for secrets |
| `internal/ssh/` | SSH certificates, `ssh`/`exec`/`copy` command construction |
| `internal/sync/` | `rumpty sync` and its background daemon |
| `internal/vm/`, `internal/workspace/` | VM and workspace operations |
| `internal/term/` | Output: tables, colours, spinners, prompts |

Keep commands thin: parse input in `cmd/`, do the work in `internal/`.

## Sign your commits (DCO)

Every commit must be signed off under the [Developer Certificate of Origin](https://developercertificate.org/): a statement that you wrote the change or otherwise have the right to submit it under the project's license. Add the sign-off with `-s`:

```sh
git commit -s -m "Add --json to vm ls"
```

This appends `Signed-off-by: Your Name <you@example.com>`, which must match the commit author. A check on every pull request enforces it. To sign off commits you already made:

```sh
git rebase --signoff origin/master
git push --force-with-lease
```

## Pull requests

- One change per pull request. Fill in the pull request template: a check fails until it has a summary, at least one `/kind` line, how you tested it, and a release note (`NONE` if nothing user-facing changed; a `/kind breaking` change always needs one).
- Rebase on the latest `master` before you open it.
- Commit messages: a short imperative subject, for example `Add --json to vm ls`.
- Only add comments that explain something the code cannot.
- Update `README.md` when you change user-facing behaviour.
- CI must pass. A maintainer reviews every pull request.

## License

By contributing, you agree that your contributions are licensed under the [Apache License 2.0](LICENSE), the same license as the project.
