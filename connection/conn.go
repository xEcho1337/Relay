package connection

import (
	"log/slog"
	"net"
	"time"

	"relay/config"
	"relay/network/packets"
	"relay/sdk"
)

const (
	// All packets up to and including the first status/login packet must
	// arrive within this window. It is cleared once the session enters relay.
	handshakeTimeout = 15 * time.Second
	backendTimeout   = 5 * time.Second
)

func HandleConnection(client net.Conn, cfg *config.Config) {
	defer client.Close()

	remoteAddr := ""
	if client.RemoteAddr() != nil {
		remoteAddr = client.RemoteAddr().String()
	}
	entry := DefaultManager.Add(remoteAddr)
	defer DefaultManager.Remove(entry.ID)

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
	switch hs.NextState {
	case 1:
		DefaultManager.SetState(entry.ID, sdk.StateStatus)
	case 2:
		DefaultManager.SetState(entry.ID, sdk.StateLogin)
	case 3:
		DefaultManager.SetState(entry.ID, sdk.StateTransfer)
	}

	slog.Info("handshake",
		"addr", client.RemoteAddr(), "host", hs.Address, "port", hs.Port,
		"protocol", hs.Protocol, "next", hs.NextState)

	// Status has no LoginStart packet; forward the handshake and relay.
	if hs.NextState == 1 {
		_ = client.SetReadDeadline(time.Time{})
		forwardAndRelay(client, cfg, hs, nil, "", entry.ID)
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

	_ = client.SetReadDeadline(time.Time{})
	forwardAndRelay(client, cfg, hs, login, playerUUID, entry.ID)
}
