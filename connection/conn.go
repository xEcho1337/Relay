package connection

import (
	"log/slog"
	"net"
	"time"

	"relay/config"
	"relay/network/packets"
)

const (
	// All packets up to and including the first status/login packet must
	// arrive within this window. It is cleared once the session enters relay.
	handshakeTimeout = 15 * time.Second
	backendTimeout   = 5 * time.Second
)

func HandleConnection(client net.Conn, cfg *config.Config) {
	defer client.Close()

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

	slog.Info("handshake",
		"addr", client.RemoteAddr(), "host", hs.Address, "port", hs.Port,
		"protocol", hs.Protocol, "next", hs.NextState)

	// Status has no LoginStart packet; forward the handshake and relay.
	if hs.NextState == 1 {
		_ = client.SetReadDeadline(time.Time{})
		forwardAndRelay(client, cfg, hs, nil, "")
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
		enc, uuid, ok := authenticate(client, login.Username)
		if !ok {
			return
		}
		client = enc
		playerUUID = uuid
	}

	_ = client.SetReadDeadline(time.Time{})
	forwardAndRelay(client, cfg, hs, login, playerUUID)
}
