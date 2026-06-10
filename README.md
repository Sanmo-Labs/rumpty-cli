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
