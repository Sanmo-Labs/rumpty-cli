# rumpty-cli

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
