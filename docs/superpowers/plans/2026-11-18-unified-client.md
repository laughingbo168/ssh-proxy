# Unified SSH/SFTP Client Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `connect` with shared connection parameters and a post-connect Shell/SFTP menu.

**Architecture:** The client builds one `ssh.ClientConfig` from `-host`, `-p`, `-u`, and `-pwd`; it verifies host keys through `known_hosts`, establishes one client connection, then repeatedly offers local Shell, SFTP, and Exit actions.

**Tech Stack:** Go, `golang.org/x/crypto/ssh`, `github.com/pkg/sftp`.

---

### Task 1: Parse and validate client options

**Files:**
- Create: `internal/client/options.go`
- Create: `internal/client/options_test.go`
- Modify: `cmd/ssh-proxy/main.go`

- [ ] Write tests requiring non-empty host, user, and password, and a port in 1–65535.
- [ ] Implement `Options` and `Validate` with precise English errors.
- [ ] Add `connect -host HOST -p PORT -u USER -pwd PASSWORD` to the CLI and run `go test ./internal/client`.

### Task 2: Verify host keys and establish one SSH client

**Files:**
- Create: `internal/client/connect.go`
- Create: `internal/client/connect_test.go`

- [ ] Write a test that rejects an unknown changed host key and accepts a known matching key.
- [ ] Implement `knownhosts.New(~/.ssh/known_hosts)` and first-connect fingerprint confirmation before appending a verified key.
- [ ] Build `ssh.ClientConfig` with password authentication and run the connection tests.

### Task 3: Add unified interactive menu

**Files:**
- Create: `internal/client/menu.go`
- Create: `internal/client/menu_test.go`

- [ ] Write menu tests for `1` Shell, `2` SFTP, and `0` Exit.
- [ ] Implement PTY Shell bridging and a small SFTP REPL supporting `ls`, `cd`, `pwd`, `get`, `put`, and `exit`.
- [ ] Return to the menu after either session closes; run `go test ./...`.

### Task 4: Document and build all binaries

**Files:**
- Modify: `README.md`

- [ ] Document the single `connect` command and menu choices.
- [ ] Run `go vet ./...`, `go test ./...`, and cross-compile Linux amd64 plus macOS arm64/amd64 artifacts.
