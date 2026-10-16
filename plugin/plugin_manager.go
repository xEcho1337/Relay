package plugin

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"relay/abi"
	"relay/sdk"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

var WasmContext context.Context
var Runtime wazero.Runtime

func Init() {
	WasmContext = context.Background()
	Runtime = wazero.NewRuntime(WasmContext)
	_, _ = wasi_snapshot_preview1.Instantiate(WasmContext, Runtime)
}

// WASMPlugin is a loaded plugin module connected to the Relay event bus.
type WASMPlugin struct {
	path          string
	module        api.Module
	alloc         api.Function
	handleMessage api.Function
	free          api.Function
	memory        api.Memory
}

// LoadWASM loads a plugin implementing the versioned Relay ABI and subscribes
// it to login events. The module must export plugin_alloc,
// plugin_handle_message, plugin_free, and memory.
func LoadWASM(ctx context.Context, path string, bus *sdk.Bus) (*WASMPlugin, error) {
	if Runtime == nil {
		Init()
	}

	wasm, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read plugin %q: %w", path, err)
	}
	if len(wasm) < 8 || string(wasm[:4]) != "\x00asm" {
		return nil, fmt.Errorf("plugin %q is not a WebAssembly module; build with GOOS=wasip1 GOARCH=wasm", path)
	}

	compiled, err := Runtime.CompileModule(ctx, wasm)
	if err != nil {
		return nil, fmt.Errorf("compile plugin %q: %w", path, err)
	}

	module, err := Runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("instantiate plugin %q: %w", path, err)
	}
	if initialize := module.ExportedFunction("_initialize"); initialize != nil {
		if _, err := initialize.Call(ctx); err != nil {
			_ = module.Close(ctx)
			return nil, fmt.Errorf("initialize plugin %q: %w", path, err)
		}
	}

	plugin := &WASMPlugin{
		path:          path,
		module:        module,
		alloc:         module.ExportedFunction("plugin_alloc"),
		handleMessage: module.ExportedFunction("plugin_handle_message"),
		free:          module.ExportedFunction("plugin_free"),
		memory:        module.Memory(),
	}
	if plugin.alloc == nil || plugin.handleMessage == nil || plugin.free == nil || plugin.memory == nil {
		_ = module.Close(ctx)
		return nil, fmt.Errorf("plugin %q does not implement Relay ABI v%d", path, sdk.ABIVersion)
	}
	// Relay never invokes _start, so a plain command module would load but
	// its init handlers would never run and every event would fail open to
	// Allow. Reject that upfront instead of running unprotected.
	if module.ExportedFunction("_initialize") == nil {
		_ = module.Close(ctx)
		return nil, fmt.Errorf("plugin %q does not export _initialize; build it as a WASI reactor with GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared", path)
	}

	bus.Login.SubscribeMutating(func(event *sdk.LoginEvent) (sdk.Decision, string) {
		request, encodeErr := abi.EncodeLogin(event.ConnectionID, event.Username, event.UUID, event.Backend)
		decision, payload, username, uuid, backend, changed, err := plugin.handle(ctx, request, encodeErr)
		if err != nil {
			slog.Warn("plugin login failed, allowing", "plugin", plugin.path, "err", err)
			return sdk.Allow, ""
		}
		if event != nil && changed != 0 {
			// Apply through the setters so change tracking propagates to
			// proxy-side handlers inspecting ChangedFields.
			if changed&uint8(sdk.LoginUsername) != 0 {
				event.SetUsername(username)
			}
			if changed&uint8(sdk.LoginUUID) != 0 {
				event.SetUUID(uuid)
			}
			if changed&uint8(sdk.LoginBackend) != 0 {
				event.SetBackend(backend)
			}
		}
		return decision, payload
	})
	bus.Handshake.Subscribe(func(event sdk.HandshakeEvent) (sdk.Decision, string) {
		request, encodeErr := abi.EncodeHandshake(event.ConnectionID, event.Host, event.ClientIP, event.Protocol, event.Port, uint16(event.NextState))
		decision, payload, _, _, _, _, err := plugin.handle(ctx, request, encodeErr)
		if err != nil {
			slog.Warn("plugin handshake failed, allowing", "plugin", plugin.path, "err", err)
			return sdk.Allow, ""
		}
		return decision, payload
	})
	bus.Disconnect.Subscribe(func(event sdk.DisconnectEvent) (sdk.Decision, string) {
		request, encodeErr := abi.EncodeDisconnect(event.ConnectionID, event.Username, event.UUID, event.Reason)
		decision, payload, _, _, _, _, err := plugin.handle(ctx, request, encodeErr)
		if err != nil {
			slog.Warn("plugin disconnect failed, allowing", "plugin", plugin.path, "err", err)
			return sdk.Allow, ""
		}
		return decision, payload
	})

	return plugin, nil
}

// EnsureDirectory creates the plugin directory (including parents) when it
// does not exist, so operators can drop .wasm files in right after startup.
func EnsureDirectory(directory string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create plugin directory %q: %w", directory, err)
	}
	return nil
}

// LoadDirectory loads every .wasm plugin in a directory in filename order.
// A missing directory is treated as an empty plugin set.
func LoadDirectory(ctx context.Context, directory string, bus *sdk.Bus) ([]*WASMPlugin, error) {
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read plugin directory %q: %w", directory, err)
	}

	plugins := make([]*WASMPlugin, 0, len(entries))
	for _, entry := range entries {
		slog.Info("loading " + entry.Name())

		if entry.IsDir() || filepath.Ext(entry.Name()) != ".wasm" {
			continue
		}
		loaded, err := LoadWASM(ctx, filepath.Join(directory, entry.Name()), bus)
		if err != nil {
			for _, loadedPlugin := range plugins {
				_ = loadedPlugin.Close(ctx)
			}
			return nil, err
		}
		plugins = append(plugins, loaded)
	}
	return plugins, nil
}

func (p *WASMPlugin) handle(ctx context.Context, request []byte, err error) (sdk.Decision, string, string, string, string, uint8, error) {
	if err != nil {
		return sdk.Allow, "", "", "", "", 0, err
	}

	results, err := p.alloc.Call(ctx, uint64(len(request)))
	if err != nil {
		return sdk.Allow, "", "", "", "", 0, fmt.Errorf("allocate plugin request: %w", err)
	}
	if len(results) != 1 {
		return sdk.Allow, "", "", "", "", 0, fmt.Errorf("allocate plugin request: expected one result")
	}
	requestPtr := uint32(results[0])
	if ok := p.memory.Write(requestPtr, request); !ok {
		return sdk.Allow, "", "", "", "", 0, fmt.Errorf("write plugin request")
	}

	responseResults, err := p.handleMessage.Call(ctx, uint64(requestPtr), uint64(len(request)))
	if err != nil {
		return sdk.Allow, "", "", "", "", 0, fmt.Errorf("handle plugin login: %w", err)
	}
	if len(responseResults) != 1 {
		return sdk.Allow, "", "", "", "", 0, fmt.Errorf("handle plugin login: expected one result")
	}
	packed := responseResults[0]
	responsePtr := uint32(packed >> 32)
	responseLen := uint32(packed)
	response, ok := p.memory.Read(responsePtr, responseLen)
	if !ok {
		return sdk.Allow, "", "", "", "", 0, fmt.Errorf("read plugin response")
	}
	copyResponse := append([]byte(nil), response...)
	_, _ = p.free.Call(ctx, uint64(requestPtr), uint64(len(request)))
	_, _ = p.free.Call(ctx, uint64(responsePtr), uint64(responseLen))
	decision, payload, username, uuid, backend, changed, err := abi.DecodeLoginResponse(copyResponse)
	if err == nil && len(request) >= 4 && request[2] == byte(abi.Login) {
		// Login mutations are applied by the caller through the event pointer.
		_ = username
		_ = uuid
		_ = backend
		_ = changed
	}
	return sdk.Decision(decision), payload, username, uuid, backend, changed, err
}

// Close releases the plugin module.
func (p *WASMPlugin) Close(ctx context.Context) error {
	if p == nil || p.module == nil {
		return nil
	}
	return p.module.Close(ctx)
}
