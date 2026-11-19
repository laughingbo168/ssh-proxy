# ssh-proxy

[English](#english) | [中文](#中文)

---

## 中文

`ssh-proxy` 是一个可部署在目标机器自定义 TCP 端口上的轻量 SSH 服务。它支持交互式 Shell、SFTP 文件传输和远程命令执行，适用于目标机器的 22 端口不可直连、但可开放其他端口的场景。

### 功能

- 自定义监听端口
- 用户名和密码认证
- 指定 Linux 用户执行 Shell、SFTP 和远程命令
- 标准 SSH、SFTP 客户端兼容
- 内置 `connect` 客户端，支持 SSH/SFTP 与密码参数
- 服务端主机密钥持久化，客户端通过 `known_hosts` 校验
- 可选来源 IP 白名单

### 编译

需要 Go 1.25+，在项目根目录执行：

```sh
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o ssh-proxy-linux-amd64 ./cmd/ssh-proxy

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o ssh-proxy-linux-arm64 ./cmd/ssh-proxy

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -o ssh-proxy-darwin-arm64 ./cmd/ssh-proxy
```

> 该服务依赖 Unix 的 PTY 伪终端与 setuid 切换用户，仅支持 Linux/macOS 部署，无法在 Windows 上编译运行。

### 服务端启动

Linux x86_64 示例：

```sh
chmod +x ssh-proxy-linux-amd64
./ssh-proxy-linux-amd64 serve -p 2222 -u user -pwd 'your-password'
```

`-u user` 指定实际运行 Shell、SFTP 和命令的 Linux 用户。若启动账号与该用户不同，需使用 `sudo`。`-pwd` 与 `-password` 等效。

未使用配置文件时，服务端默认监听 `0.0.0.0:2222`，在当前目录创建 `host_ed25519` 主机私钥，并输出无来源 IP 白名单警告。生产环境请通过防火墙或安全组仅允许可信 IP 访问端口。

### 客户端使用

```sh
# 标准 SSH Shell
ssh user@TARGET_IP -p 2222

# 标准 SFTP
sftp -P 2222 user@TARGET_IP

# 标准 SSH 远程命令
ssh user@TARGET_IP -p 2222 'uname -a'

# 内置客户端：SSH Shell
./ssh-proxy-darwin-arm64 connect -protocol ssh -h TARGET_IP -p 2222 -u user -pwd 'your-password'

# 内置客户端：SFTP
./ssh-proxy-darwin-arm64 connect -protocol sftp -h TARGET_IP -p 2222 -u user -pwd 'your-password'

# 内置客户端：远程命令
./ssh-proxy-darwin-arm64 connect -protocol ssh -h TARGET_IP -p 2222 -u user -pwd 'your-password' -c 'uname -a'
```

首次连接时，请核对并确认服务端主机密钥指纹。不要随意删除或更换 `host_ed25519`，否则客户端会提示主机密钥变化。

### 安全说明

密码参数会出现在启动服务的进程参数中，仅建议临时使用。长期部署请限制目标端口的来源 IP，并妥善保护 `host_ed25519`：

```sh
chmod 600 host_ed25519
```

---

## English

`ssh-proxy` is a lightweight SSH service for a custom TCP port on the target host. It supports interactive shells, SFTP file transfer, and remote command execution when port 22 is not directly reachable but another port can be opened.

### Features

- Configurable listening port
- Username and password authentication
- Run Shell, SFTP, and remote commands as a selected Linux user
- Compatible with standard SSH and SFTP clients
- Built-in `connect` client with SSH/SFTP and password parameters
- Persistent server host key with client-side `known_hosts` verification
- Optional source IP allowlist

### Build

Requires Go 1.25+. Run in the project root:

```sh
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o ssh-proxy-linux-amd64 ./cmd/ssh-proxy

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o ssh-proxy-linux-arm64 ./cmd/ssh-proxy

# macOS arm64
GOOS=darwin GOARCH=arm64 go build -o ssh-proxy-darwin-arm64 ./cmd/ssh-proxy
```

> The service relies on Unix PTY pseudo-terminals and setuid user switching, so it supports Linux/macOS deployment only and cannot be built or run on Windows.

### Start the server

Linux x86_64 example:

```sh
chmod +x ssh-proxy-linux-amd64
./ssh-proxy-linux-amd64 serve -p 2222 -u user -pwd 'your-password'
```

`-u user` selects the Linux account used for shells, SFTP, and commands. Use `sudo` when the account differs from the account starting the service. `-pwd` is an alias for `-password`.

Without a configuration file, the server listens on `0.0.0.0:2222`, creates `host_ed25519` in the current directory, and warns that no source IP allowlist is configured. Restrict the port to trusted IPs through a firewall or security group in production.

### Connect

```sh
# Standard SSH shell
ssh user@TARGET_IP -p 2222

# Standard SFTP
sftp -P 2222 user@TARGET_IP

# Standard SSH remote command
ssh user@TARGET_IP -p 2222 'uname -a'

# Built-in client: SSH shell
./ssh-proxy-darwin-arm64 connect -protocol ssh -h TARGET_IP -p 2222 -u user -pwd 'your-password'

# Built-in client: SFTP
./ssh-proxy-darwin-arm64 connect -protocol sftp -h TARGET_IP -p 2222 -u user -pwd 'your-password'

# Built-in client: remote command
./ssh-proxy-darwin-arm64 connect -protocol ssh -h TARGET_IP -p 2222 -u user -pwd 'your-password' -c 'uname -a'
```

Verify the server host-key fingerprint on the first connection. Do not casually delete or rotate `host_ed25519`, otherwise clients will report a changed host key.

### Security

The password argument is visible in the service process arguments and is intended for temporary use only. For long-running deployments, restrict the target port to trusted IPs and protect the server host key:

```sh
chmod 600 host_ed25519
```
