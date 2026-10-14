package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	key     *rsa.PrivateKey
	keyOnce sync.Once
	keyErr  error
	pubDER  []byte
)

// Key returns the proxy RSA key. 2048 bits remain compatible with Minecraft
// 1.7+ clients; RSA responses are 256 bytes long.
func Key() (*rsa.PrivateKey, error) {
	keyOnce.Do(func() {
		key, keyErr = rsa.GenerateKey(rand.Reader, 2048)
		if keyErr != nil {
			return
		}
		pubDER, keyErr = x509.MarshalPKIXPublicKey(&key.PublicKey)
	})
	return key, keyErr
}

func PublicKeyDER() ([]byte, error) {
	if _, err := Key(); err != nil {
		return nil, err
	}
	return pubDER, nil
}

func RandomToken() ([]byte, error) {
	tok := make([]byte, 16)
	if _, err := rand.Read(tok); err != nil {
		return nil, err
	}
	return tok, nil
}

// ServerHash computes the session hash expected by Mojang:
// SHA1(serverId + secret + publicKeyDER) in Java hexadecimal format
// (two's complement, lowercase, no leading zeroes, '-' when negative).
func ServerHash(serverID string, secret, pub []byte) string {
	h := sha1.New()
	h.Write([]byte(serverID))
	h.Write(secret)
	h.Write(pub)
	sum := h.Sum(nil)

	neg := sum[0]&0x80 != 0
	var n *big.Int
	if neg {
		inv := make([]byte, len(sum))
		for i, b := range sum {
			inv[i] = ^b
		}
		n = new(big.Int).SetBytes(inv)
		n.Add(n, big.NewInt(1))
	} else {
		n = new(big.Int).SetBytes(sum)
	}

	s := strings.TrimLeft(n.Text(16), "0")
	if s == "" {
		s = "0"
	}
	if neg {
		s = "-" + s
	}
	return s
}

type profile struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// HasJoined asks the Mojang session server. 200 = account premium
func HasJoined(username, serverHash string) (undashedID string, err error) {
	u := "https://sessionserver.mojang.com/session/minecraft/hasJoined?username=" +
		url.QueryEscape(username) + "&serverId=" + url.QueryEscape(serverHash)

	resp, err := httpClient.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("mojang hasJoined: %s", resp.Status)
	}

	var p profile
	if err := json.Unmarshal(body, &p); err != nil {
		return "", err
	}

	if len(p.ID) != 32 {
		return "", fmt.Errorf("mojang: bad profile id %q", p.ID)
	}

	return p.ID, nil
}

// DashUUID converts "32hex" to "8-4-4-4-12" format.
func DashUUID(undashed string) string {
	return undashed[0:8] + "-" + undashed[8:12] + "-" + undashed[12:16] +
		"-" + undashed[16:20] + "-" + undashed[20:32]
}
