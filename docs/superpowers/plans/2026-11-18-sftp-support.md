# SFTP Support Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add standard SFTP access using the existing SSH username, password, and Linux user identity.

**Architecture:** A `subsystem` request named `sftp` is handled by an embedded `github.com/pkg/sftp` server on the SSH session channel. Other subsystem requests remain rejected; shell, SCP, exec, and forwarding behavior remains unchanged.

**Tech Stack:** Go, `golang.org/x/crypto/ssh`, `github.com/pkg/sftp`.

---

### Task 1: Add failing SFTP subsystem coverage

**Files:**
- Create: `internal/server/sftp_test.go`
- Modify: `internal/server/session.go`

- [ ] **Step 1: Write a failing SFTP request test**

```go
func TestSFTPSubsystemIsAccepted(t *testing.T) {
    client := dialTestServer(t)
    sftpClient, err := sftp.NewClient(client)
    if err != nil { t.Fatal(err) }
    defer sftpClient.Close()
    if _, err := sftpClient.ReadDir("."); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 2: Run test**

Run: `go test ./internal/server -run TestSFTPSubsystemIsAccepted -v`

Expected: fail because `subsystem` requests are rejected.

### Task 2: Implement embedded SFTP server

**Files:**
- Create: `internal/server/sftp.go`
- Modify: `internal/server/session.go`
- Modify: `go.mod`

- [ ] **Step 1: Add `serveSFTP`**

```go
func serveSFTP(channel ssh.Channel) {
    server, err := sftp.NewServer(channel)
    if err != nil { _ = channel.Close(); return }
    _ = server.Serve()
    _ = server.Close()
    _ = channel.Close()
}
```

- [ ] **Step 2: Dispatch only the `sftp` subsystem**

Decode the request payload using `ssh.Unmarshal` into `struct { Name string }`. Reply success and invoke `serveSFTP(channel)` only when `Name == "sftp"`; reply false otherwise. A session can start either SFTP or a shell, never both.

- [ ] **Step 3: Run tests**

Run: `GOPROXY=https://goproxy.cn,direct go get github.com/pkg/sftp && go test ./internal/server -v`

Expected: standard SFTP request succeeds and unknown subsystems are rejected.

### Task 3: Document and build

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Add usage**

Document: `sftp -P 2222 user@TARGET_IP`, noting that it uses the same password and that filesystem permissions match the service Linux user.

- [ ] **Step 2: Verify and compile**

Run: `go test ./... && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o ssh-proxy-linux-amd64 ./cmd/ssh-proxy`

Expected: all tests pass and the Linux binary is produced.
