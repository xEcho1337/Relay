package connection

import (
	"log/slog"
	"net"
	"time"

	"relay/config"
	"relay/network/packets"
	"relay/plugin"
	"relay/protocol"
	"relay/sdk"
)

const (
	// All packets up to and including the first status/login packet must
	// arrive within this window. It is cleared once the session enters relay.
	handshakeTimeout = 15 * time.Second
	backendTimeout   = 5 * time.Second
)

func HandleConnection(client net.Conn, cfg *config.Config, bus *plugin.Bus) {
	defer client.Close()

	remoteAddr := ""
	if client.RemoteAddr() != nil {
		remoteAddr = client.RemoteAddr().String()
	}
	entry := DefaultManager.Add(remoteAddr)
	backendOverride := ""
	discUsername := ""
	discUUID := ""
	discReason := ""
	defer func() {
		if discUsername != "" {
			bus.Disconnect.Emit(sdk.DisconnectEvent{
				ConnectionID: entry.ID,
				Username:     discUsername,
				UUID:         discUUID,
				Reason:       discReason,
			})
		}
		DefaultManager.Remove(entry.ID)
	}()

	if err := client.SetReadDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		slog.Error("set deadline", "addr", client.RemoteAddr(), "err", err)
		return
	}

	// The first packet is always the handshake.
	hs, err := packets.ReadHandshake(client)
	if err != nil {
		slog.Error("handshake failed", "addr", client.RemoteAddr(), "err", err)
		return
	}

	DefaultManager.SetProtocol(entry.ID, hs.Protocol)
	connState := protocol.StateHandshake
	switch hs.NextState {
	case 1:
		connState = protocol.StateStatus
	case 2:
		connState = protocol.StateLogin
	case 3:
		connState = protocol.StateTransfer
	}
	DefaultManager.SetState(entry.ID, connState)

	if decision, payload := bus.Handshake.Emit(sdk.HandshakeEvent{
		ConnectionID: entry.ID,
		Protocol:     uint32(hs.Protocol),
		Host:         hs.Address,
		Port:         hs.Port,
		NextState:    sdk.HandshakeNextState(hs.NextState),
		ClientIP:     clientIP(client),
	}); decision == sdk.Deny {
		denyConnection(client, connState, hs.Protocol, payload)
		return
	} else if decision == sdk.OverrideBackend {
		backendOverride = payload
	}

	slog.Info("handshake",
		"addr", client.RemoteAddr(), "host", hs.Address, "port", hs.Port,
		"protocol", hs.Protocol, "next", hs.NextState)

	// Status has no LoginStart packet; forward the handshake and relay.
	if hs.NextState == 1 {
		_ = client.SetReadDeadline(time.Time{})
		forwardAndRelay(client, cfg, hs, nil, "", entry.ID, backendOverride)
		return
	}

	if hs.NextState != 2 && hs.NextState != 3 {
		slog.Warn("unknown next state", "next", hs.NextState, "addr", client.RemoteAddr())
		return
	}

	// Login and transfer require LoginStart as the next packet.
	if err := client.SetReadDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		slog.Error("set deadline", "addr", client.RemoteAddr(), "err", err)
		return
	}
	login, err := packets.ReadLoginStart(client)
	if err != nil {
		slog.Error("login start", "addr", client.RemoteAddr(), "err", err)
		return
	}

	slog.Info("login start", "addr", client.RemoteAddr(), "username", login.Username)
	discUsername = login.Username

	playerUUID := offlineUUID(login.Username)
	if cfg.Network.OnlineMode {
		enc, uuid, ok := authenticate(client, login.Username, cfg, hs.Protocol)
		if !ok {
			return
		}
		client = enc
		playerUUID = uuid
	}

	DefaultManager.SetPlayer(entry.ID, sdk.Player{
		Username:    login.Username,
		UUID:        playerUUID,
		ClientIP:    clientIP(client),
		ConnectedAt: uint64(time.Now().UnixMilli()),
	})
	discUsername = login.Username
	discUUID = playerUUID

	defaultBackend, err := resolveBackend(cfg, hs.Address)
	if err != nil {
		slog.Error("no backend for host", "host", hs.Address, "err", err)
		_ = writeMiniDisconnect(client, cfg.Messages.NoServersConfigured)
		return
	}

	if decision, payload := bus.Login.Emit(sdk.LoginEvent{
		ConnectionID: entry.ID,
		Username:     login.Username,
		UUID:         playerUUID,
		Backend:      defaultBackend,
	}); decision == sdk.Deny {
		discReason = payload
		loginState := protocol.StateLogin
		if hs.NextState == 3 {
			loginState = protocol.StateTransfer
		}
		denyConnection(client, loginState, hs.Protocol, payload)
		return
	} else if decision == sdk.OverrideBackend {
		backendOverride = payload
	}

	_ = client.SetReadDeadline(time.Time{})
	forwardAndRelay(client, cfg, hs, login, playerUUID, entry.ID, backendOverride)
}

func denyConnection(client net.Conn, state protocol.State, proto int, reason string) {
	_, _ = packets.WriteDisconnect(client, state, proto, reason)
	slog.Info("connection denied", "addr", client.RemoteAddr(), "reason", reason)
}
