# Relay

Minimal and performant Minecraft proxy written in Go.


The project is intentionally compact today. The planned extension points are a versioned plugin API, lifecycle hooks, 
backend and player events, and a container image for reproducible deployments.

## Table of Contents

- [Requirements](#requirements)
- [Install](#install)
- [Usage](#usage)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Roadmap](#roadmap)
- [Development](#development)
- [Contributing](#contributing)
- [License](#license)

## Requirements

- Go 1.27 or newer
- A Minecraft backend server compatible with the protocol version used by clients

## Install

Build Relay from source:

```sh
go build -o relay .
```

The GitHub Actions workflow produces binaries for Linux, macOS, and Windows on
both `amd64` and `arm64`.

## Usage

Run Relay from the directory where its configuration should live:

```sh
./relay
```

If `config.yml` does not exist, Relay creates it with safe starter values and then loads it. Existing configuration 
files are never overwritten automatically.

## Configuration

Relay reads `config.yml` from the current working directory. 

When `ip-forwarding` is enabled, the backend must support BungeeCord forwarding and have its corresponding forwarding 
setting enabled.

With `online-mode` enabled, Relay authenticates the player before forwarding the connection; the backend is expected to
run in offline mode and trust only the proxy network path.

The `id` value is matched against the host sent in the Minecraft handshake. If there is exactly one configured backend,
it is used as the fallback target.

## Architecture

Relay has a short packet-aware front end followed by a transparent byte relay.
It reads the handshake to select a backend and, for login connections, reads
LoginStart and handles authentication when online mode is enabled. After setup,
it forwards traffic in both directions without parsing subsequent Minecraft
packets. Status requests follow the same transparent path immediately after
the handshake.

## Roadmap

- [X] Define a stable, versioned plugin API.
- [X] Add plugin discovery, configuration, lifecycle hooks, and isolation boundaries.
- [ ] Expose backend selection, connection, authentication, and disconnect events.
- [ ] Provide a first-party Docker image with a mounted configuration directory.

## Development

Run the test suite:

```sh
go test ./...
```

Format Go files before opening a change:

```sh
gofmt -w .
```

## Plugin development

Plugin authors use `relay/sdk`, not the WASM ABI directly. A plugin registers
typed handlers through `sdk.Registry`:

```go
func Register(registry *sdk.Registry) {
    registry.OnLogin(func(event sdk.LoginEvent) sdk.Result {
        if event.Username == "TestUsername" {
            return sdk.Result{Decision: sdk.Deny, Payload: "Username not valid"}
        }
        return sdk.Result{Decision: sdk.Allow}
    })
}
```

Make sure to compile your program with WASM:
```bash
GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared -o plugin.wasm .
```

Drop your `.wasm` files inside the `plugins` folder generated at startup by Relay.

## Contributing

Open an issue for bugs or design proposals. For code changes, include focused tests where behavior changes, 
keep public APIs small, and explain compatibility implications for Minecraft 1.7 through the latest supported release.

## License

Relay is licensed under the GNU General Public License v3.0.
