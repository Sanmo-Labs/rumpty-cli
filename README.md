# rumpty-cli

Manage workspaces, VMs, and storage on [Rumpty Cloud](https://rumptycloud.com) from your terminal.

## Install

Install the latest release with the install script:

```sh
curl -fsSL https://get.rumptycloud.com | sh
```

The script downloads the release for your OS and architecture, verifies its checksum, and installs the `rumpty` binary to `/usr/local/bin` (falling back to `~/.local/bin` when that directory is not writable).

Optional overrides:

```sh
# Install a specific version
curl -fsSL https://get.rumptycloud.com | RUMPTY_VERSION=v0.0.3 sh

# Install to a custom directory
curl -fsSL https://get.rumptycloud.com | RUMPTY_INSTALL_DIR="$HOME/.local/bin" sh
```

Verify the install:

```sh
rumpty --version
```

## Quick Start

Sign in through your browser (stores a session for later commands):

```sh
rumpty login
```

SSH into a workspace VM:

```sh
rumpty ssh <vm>
```

Back up a local folder to object storage — the bucket is created automatically on first run, and repeat runs only transfer changes:

```sh
rumpty sync ~/sales-reports my-backups
```

Add `--watch` to keep syncing as files change, or `--daemon` to do the same in a background process (`rumpty sync status` / `rumpty sync stop` manage it). See the [docs](https://docs.rumptycloud.com/cli/introduction) for every command and flag.

## Commands

| Command | Description |
|---------|-------------|
| `login` / `logout` | Authenticate with Rumpty / remove the local session |
| `ssh` | Open an SSH session to a workspace VM |
| `sync` | Sync a local folder to a bucket in object storage |
| `copy` (alias `cp`) | Copy files between your machine and a VM |
| `exec` | Run a non-interactive command on a VM |
| `expose` / `unexpose` | Add or remove a public URL for a VM service |
| `vm` | Manage workspace VMs: `ls`, `start`, `stop`, `reboot`, `delete` |
| `workspaces` | List workspaces you can access |
| `completion` | Generate shell autocompletion script |

Use `rumpty [command] --help` for details on any command.

## Expose a VM Service

Expose a service running inside a VM with a public HTTPS URL:

```sh
rumpty expose <vm> --port <port> --name <name>
```

List exposed URLs for a VM:

```sh
rumpty vm expose ls <vm>
```

Remove an exposed URL:

```sh
rumpty unexpose <vm> --name <name>
```

The service inside the VM must listen on `0.0.0.0:<port>` so the VM network can reach it. A service bound only to `127.0.0.1:<port>` accepts connections from inside the VM itself, but not from the Rumpty HTTP route.

## Shell Completion

`rumpty` ships completion for subcommands and flags, plus dynamic completion of
live VM names and workspace slugs (`rumpty ssh <TAB>`, `rumpty vm stop <TAB>`,
`rumpty --ws <TAB>`).

Generate and install the completion script for your shell:

```sh
mkdir -p ~/.oh-my-zsh/completions
rumpty completion zsh > ~/.oh-my-zsh/completions/_rumpty
rm -f ~/.zcompdump*   # bust the completion cache once after adding a new script

rumpty completion zsh > "${fpath[1]}/_rumpty"

rumpty completion bash | sudo tee /etc/bash_completion.d/rumpty > /dev/null

rumpty completion fish > ~/.config/fish/completions/rumpty.fish
```

Restart your shell (or `exec zsh`) afterwards. Run `rumpty completion --help`
for per-shell details.

Dynamic VM/workspace completion calls the Rumpty API as you press `<TAB>`, so it
requires authentication (`rumpty login` or `$RUMPTY_API_KEY`). VM-name
completion also needs a workspace: it uses `--ws`/`--workspace`,
`$RUMPTY_WORKSPACE`, or your default workspace when none is set. If the API is
unreachable, completion fails quietly (no suggestions) instead of blocking the
prompt.
