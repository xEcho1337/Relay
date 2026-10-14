package connection

import "testing"

func TestForwardedAddressDropsClientInjectedFields(t *testing.T) {
	got := forwardedAddress("lobby\x00forged-ip\x00forged-uuid", "192.0.2.10", "real-uuid")
	want := "lobby\x00192.0.2.10\x00real-uuid"
	if got != want {
		t.Fatalf("forwarded address = %q, want %q", got, want)
	}
}
