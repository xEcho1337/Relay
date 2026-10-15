package connection

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/subtle"
	"log/slog"
	"net"
	"relay/config"
	"time"

	"relay/auth"
	"relay/network/crypt"
	"relay/network/packets"
)

// authenticate performs the premium login flow:
// EncryptionRequest, EncryptionResponse, token verification, AES-CFB8,
// and Mojang hasJoined verification.
func authenticate(client net.Conn, username string, cfg *config.Config, protocol int) (net.Conn, string, bool) {
	key, err := auth.Key()
	if err != nil {
		slog.Error("rsa key", "err", err)
		return nil, "", false
	}

	pubDER, err := auth.PublicKeyDER()
	if err != nil {
		slog.Error("rsa pubkey", "err", err)
		return nil, "", false
	}

	token, err := auth.RandomToken()
	if err != nil {
		slog.Error("verify token", "err", err)
		return nil, "", false
	}

	if err := packets.WriteEncryptionRequest(client, &packets.EncryptionRequest{
		ServerID:    "",
		PublicKey:   pubDER,
		VerifyToken: token,
	}, protocol); err != nil {
		slog.Error("encryption request", "addr", client.RemoteAddr(), "err", err)
		return nil, "", false
	}

	if err := client.SetReadDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		slog.Error("set deadline", "addr", client.RemoteAddr(), "err", err)
		return nil, "", false
	}

	resp, err := packets.ReadEncryptionResponse(client)
	if err != nil {
		slog.Error("encryption response", "addr", client.RemoteAddr(), "err", err)
		return nil, "", false
	}

	secret, err := rsa.DecryptPKCS1v15(rand.Reader, key, resp.SharedSecretEncrypted)
	if err != nil || len(secret) != 16 {
		slog.Warn("bad shared secret", "addr", client.RemoteAddr(), "err", err)
		_ = writeMiniDisconnect(client, cfg.Messages.UsernameVerificationFailed)
		return nil, "", false
	}

	decryptedToken, err := rsa.DecryptPKCS1v15(rand.Reader, key, resp.VerifyTokenEncrypted)
	if err != nil || subtle.ConstantTimeCompare(decryptedToken, token) != 1 {
		slog.Warn("bad verify token", "addr", client.RemoteAddr())
		_ = writeMiniDisconnect(client, cfg.Messages.UsernameVerificationFailed)
		return nil, "", false
	}

	undashed, err := auth.HasJoined(username, auth.ServerHash("", secret, pubDER))
	if err != nil {
		slog.Info("hasJoined failed", "username", username, "err", err)
		_ = writeMiniDisconnect(client, cfg.Messages.UsernameVerificationFailed)
		return nil, "", false
	}

	enc, err := crypt.NewConn(client, secret)
	if err != nil {
		slog.Error("enable encryption", "addr", client.RemoteAddr(), "err", err)
		return nil, "", false
	}

	uuid := auth.DashUUID(undashed)
	slog.Info("authenticated", "username", username, "uuid", uuid)
	return enc, uuid, true
}
