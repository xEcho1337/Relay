package connection

import (
	"log/slog"
	"net"
	"strings"

	"relay/config"
	"relay/network/packets"
	"relay/protocol"

	"github.com/xEcho1337/fable/component"
)

func forwardAndRelay(client net.Conn, cfg *config.Config, hs *packets.Handshake, login *packets.LoginStart, playerUUID string, connID uint64, backendOverride string) {
	if len(cfg.Servers) == 0 {
		slog.Error("no server configured")
		_ = writeMiniDisconnect(client, cfg.Messages.NoServersConfigured)
		return
	}

	backendAddr := backendOverride
	if backendAddr == "" {
		resolved, err := resolveBackend(cfg, hs.Address)
		if err != nil {
			slog.Error("no backend for host", "host", hs.Address, "err", err)
			_ = writeMiniDisconnect(client, cfg.Messages.NoServersConfigured)
			return
		}
		backendAddr = resolved
	}
	DefaultManager.SetBackend(connID, backendAddr)

	backend, err := net.DialTimeout("tcp", backendAddr, backendTimeout)
	if err != nil {
		slog.Error("dial backend", "backend", backendAddr, "err", err)
		_ = writeMiniDisconnect(client, cfg.Messages.ServerOffline)
		return
	}

	defer backend.Close()

	if cfg.Network.IPForwarding && login != nil {
		// Only the host portion belongs in the BungeeCord forwarding field.
		// The client controls hs.Address and may otherwise inject additional
		// NUL-delimited fields into the backend handshake.
		fwd := &packets.Handshake{
			Protocol:  hs.Protocol,
			Address:   forwardedAddress(hs.Address, clientIP(client), playerUUID),
			Port:      hs.Port,
			NextState: hs.NextState,
		}
		if err := packets.WriteHandshake(backend, fwd); err != nil {
			slog.Error("forward handshake", "backend", backendAddr, "err", err)
			return
		}
		if _, err := backend.Write(login.Raw); err != nil {
			slog.Error("forward login", "backend", backendAddr, "err", err)
			return
		}
	} else {
		if _, err := backend.Write(hs.Raw); err != nil {
			slog.Error("forward handshake", "backend", backendAddr, "err", err)
			return
		}
		if login != nil {
			if _, err := backend.Write(login.Raw); err != nil {
				slog.Error("forward login", "backend", backendAddr, "err", err)
				return
			}
		}
	}

	if login != nil {
		DefaultManager.SetState(connID, protocol.StatePlay)
	}

	relay(client, backend)
}

func forwardedAddress(address, ip, playerUUID string) string {
	if i := strings.IndexByte(address, 0); i >= 0 {
		address = address[:i]
	}
	return address + "\x00" + ip + "\x00" + playerUUID
}

func writeMiniDisconnect(client net.Conn, mini string) error {
	raw, err := component.Marshal(component.ParseMiniMessage(mini))
	if err != nil {
		return packets.WriteLoginDisconnect(client, mini)
	}
	return packets.WriteLoginDisconnectJSON(client, string(raw))
}
