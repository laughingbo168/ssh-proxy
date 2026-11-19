# SSH Proxy Design

## Goal

Run a small SSH-compatible server on a target machine where port 22 is not directly reachable, but another TCP port can be opened. A local user connects using the regular `ssh` command and receives an interactive shell on that machine without providing a user password or SSH key to the proxy.

## Scope

The proxy supports an interactive shell, standard SFTP file transfer, and single remote-command execution. It is intentionally not an SSH jump host, port forwarder, or SCP server.

## Architecture

- Implement a single Go binary named `ssh-proxy` using the SSH protocol.
- The process runs under the target Linux user whose shell will be exposed.
- It listens on a configured TCP address such as `0.0.0.0:443`.
- The server enables SSH `none` authentication. The SSH username is accepted but does not select a Linux account.
- The server retains a host private key at a configured path. It generates one when absent and writes it with mode `0600`; this key provides stable host identity for clients' `known_hosts` checking.
- Authentication is restricted at the network layer by an optional CIDR allowlist. Connections outside the list are rejected. An omitted list permits every source address and emits a prominent startup warning.

Local usage is standard OpenSSH:

```sh
ssh -o PreferredAuthentications=none -p 443 arbitrary-name@target-ip
```

The client has no user private-key, password, or JumpServer dependency for this connection. The network firewall or security group must only expose the configured port to trusted source IPs.

## Configuration

Use a YAML configuration file with these fields:

```yaml
listen: "0.0.0.0:443"
host_key_path: "/var/lib/ssh-proxy/host_ed25519"
allow_cidrs:
  - "203.0.113.10/32"
shell: "/bin/zsh"
```

- `listen` is required and determines the TCP interface and port.
- `host_key_path` is required.
- `allow_cidrs` is optional. When provided, it must contain valid CIDR blocks.
- `shell` is optional; its default is the process user's login shell, with `/bin/sh` as the final fallback.

## SSH Session Behavior

The server supports only session channels for interactive terminal use:

1. It accepts PTY allocation and records the requested terminal type and initial dimensions.
2. It responds to window-change requests by resizing the PTY.
3. On a shell request it starts the configured shell attached to the PTY.
4. It copies terminal input and output between the SSH channel and PTY, forwards supported signals, and cleans up when either side exits.
5. It returns the child shell's exit status to the SSH client.

It rejects direct TCP forwarding, reverse forwarding, agent forwarding, X11 forwarding, and SCP requests. This keeps the endpoint from becoming a general-purpose tunnel.

## Remote Command Behavior

The server accepts one SSH `exec` request per session. It runs the supplied command through the configured user's shell, streams stdout and stderr to the SSH channel, and returns the real process exit status. Standard OpenSSH clients use `ssh user@target-ip -p 2222 'command'`; the local `connect` client exposes the same capability through `connect -h HOST -p PORT -u USER -pwd PASSWORD -c 'command'`. A session selects exactly one of Shell, SFTP, or `exec`.

## SFTP Session Behavior

The server accepts a `session` channel `subsystem` request only when its name is `sftp`. It starts an embedded, standards-compatible SFTP server on that channel; common clients use it as follows:

```sh
sftp -P 2222 user@target-ip
```

SFTP uses the same SSH username validation, optional password authentication, and process identity as interactive shell sessions. When `-u user` is configured, all file operations use that Linux user's UID and GID. The user can access any filesystem path for which that Linux account has normal operating-system permissions. Unknown subsystem requests are rejected in English and close only their own session.

## CLI and Operations

The binary exposes:

- `ssh-proxy init`: writes an example configuration and generates a host key if needed.
- `ssh-proxy serve -c /etc/ssh-proxy/config.yaml`: validates configuration and runs the server.
- `ssh-proxy connect -host HOST -p PORT -u USER -pwd PASSWORD`: acts as a local SSH/SFTP client. It does not require a protocol parameter.

## Local Client Behavior

The `connect` command first establishes one authenticated SSH client connection, then presents a local menu with `SSH Shell`, `SFTP`, and `Exit`. Selecting Shell opens a PTY session; selecting SFTP opens an SFTP subsystem session. Closing either mode returns to the menu so the user can choose the other mode without reentering connection parameters or credentials. The command uses the supplied host, port, username, and password only for the current process; it does not write the password to a configuration file or log.

The client verifies the server host key using `~/.ssh/known_hosts`. On its first connection to an unknown host key, it displays the fingerprint and asks for explicit confirmation before appending the verified key. A changed key is rejected and requires the user to remove or update its existing `known_hosts` entry deliberately.

The distribution includes a systemd unit that runs `serve`, restarts it after failure, and stops it gracefully on service shutdown. On `SIGTERM` or `SIGINT`, the server stops accepting new connections and closes active sessions cleanly.

Connection lifecycle logs include timestamp, source IP, requested SSH username, acceptance/rejection reason, session type, and session termination. Terminal contents, transferred file contents, passwords, and shell output are never logged.

## Errors and Security Boundaries

- Invalid YAML, invalid CIDRs, missing mandatory fields, unreadable host keys, and listener bind failures make startup fail with actionable errors.
- An unknown or malformed SSH request is rejected without affecting other clients.
- Client disconnection, server shutdown, or shell exit closes the PTY and child process to prevent orphaned sessions.
- No-authentication access is safe only when the listener is protected by an IP allowlist and external firewall/security-group rules. The documentation must state this prominently.

## Tests and Acceptance

Automated tests cover configuration validation, CIDR matching, host-key persistence, authentication handshake, interactive PTY shell I/O, resize behavior, SFTP handshake and file operations, rejected forwarding/exec requests, and process cleanup.

Acceptance criteria:

1. Start the proxy on the target machine using a configured non-22 port.
2. From an allowed source, run `ssh -p <port> any-name@<target-ip>` and obtain the process user's interactive shell without a password or private key.
3. From an unallowed source, observe rejection before a shell is opened.
4. From an allowed source with valid credentials, run `sftp -P <port> user@<target-ip>` and complete a file upload/download using the selected Linux user's permissions.
5. Verify attempts at forwarding, SCP, and remote command execution are rejected.
