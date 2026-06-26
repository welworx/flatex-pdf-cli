# Installation

The tool is a single self-contained binary (no runtime dependencies).

## Option A — download a pre-built binary (recommended)

Grab the binary for your platform from the [releases page](https://github.com/welworx/flatex-pdf-cli/releases). Builds are published for:

| OS | Arch | Asset |
|---|---|---|
| macOS | Apple Silicon | `flatex-pdf-cli_darwin_arm64` |
| macOS | Intel | `flatex-pdf-cli_darwin_amd64` |
| Linux | x86-64 | `flatex-pdf-cli_linux_amd64` |
| Linux | ARM64 | `flatex-pdf-cli_linux_arm64` |
| Windows | x86-64 | `flatex-pdf-cli_windows_amd64.exe` |

Then make it executable and put it on your `$PATH`:

```bash
chmod +x flatex-pdf-cli_darwin_arm64
mv flatex-pdf-cli_darwin_arm64 /usr/local/bin/flatex-pdf-cli
```

## Option B — go install (Go 1.26+)

```bash
go install github.com/welworx/flatex-pdf-cli@latest
```

The binary lands in `$(go env GOPATH)/bin`, which should be in your `$PATH`.

## Option C — build from source

```bash
git clone https://github.com/welworx/flatex-pdf-cli.git
cd flatex-pdf-cli
go build -o flatex-pdf-cli .
```

Then add the directory to `$PATH` or move `flatex-pdf-cli` somewhere already on it (e.g., `/usr/local/bin`).

## Verify

```bash
flatex-pdf-cli -version
```

Should print the version without "command not found".
