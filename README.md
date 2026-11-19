# ssh-proxy

An SSH-compatible, interactive-shell-only proxy for a non-22 port. It does **not** authenticate users: restrict both `allow_cidrs` and the cloud security group/firewall to trusted source IPs.

## Build and deploy

```sh
go build -o ssh-proxy ./cmd/ssh-proxy
sudo install -m 0755 ssh-proxy /usr/local/bin/ssh-proxy
sudo useradd --system --create-home sshproxy
sudo mkdir -p /etc/ssh-proxy
sudo /usr/local/bin/ssh-proxy init -c /etc/ssh-proxy/config.yaml
sudo chown -R sshproxy:sshproxy /etc/ssh-proxy
sudo install -m 0644 deploy/ssh-proxy.service /etc/systemd/system/ssh-proxy.service
sudo systemctl daemon-reload
sudo systemctl enable --now ssh-proxy
```

Set `listen`, `allow_cidrs`, and `shell` in `/etc/ssh-proxy/config.yaml`; a relative `host_key_path` is resolved beside that configuration file. Open only the configured port to the same CIDRs at the host firewall/security group. The config file is optional: `ssh-proxy serve -p 2222` uses `0.0.0.0:2222`, `./host_ed25519`, and `/bin/sh`, but has no source allowlist and emits a high-risk warning.

Connect from a permitted source:

```sh
ssh -o PreferredAuthentications=none -p 2222 arbitrary-name@TARGET_IP
```

On first connection, verify and accept the displayed host-key fingerprint. The server permits an interactive PTY shell only; remote commands, SCP/SFTP, and port forwarding are rejected.

To override the port in the configuration at startup, use `-p`:

```sh
./ssh-proxy serve -c /etc/ssh-proxy/config.yaml -p 2222
```

To run each SSH shell as a specific Linux user, start the proxy as root and pass `-u`:

```sh
sudo ./ssh-proxy serve -p 2222 -u user
```

When `-u user` is used, clients must also connect as that SSH username:

```sh
ssh -tt -p 2222 user@TARGET_IP
```

SFTP uses the same username and password:

```sh
sftp -P 2222 user@TARGET_IP
```

It can access every path permitted to the service Linux user.

Require a client password with `--password` (or `--pwd`); OpenSSH will prompt for it:

```sh
sudo ./ssh-proxy serve -p 2222 -u user --password 'change-me'
ssh -tt -p 2222 user@TARGET_IP
```
