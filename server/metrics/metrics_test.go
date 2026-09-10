package metrics

import "testing"

func TestSanitizeAutoTransportProtocol(t *testing.T) {
	for _, protocol := range []string{"tcp", "kcp", "quic", "websocket", "wss"} {
		if got := SanitizeAutoTransportProtocol(protocol); got != protocol {
			t.Fatalf("expected %q to pass through, got %q", protocol, got)
		}
	}
	for _, protocol := range []string{"", "udp", "bad", "tcp\nbad"} {
		if got := SanitizeAutoTransportProtocol(protocol); got != "unknown" {
			t.Fatalf("expected %q to sanitize to unknown, got %q", protocol, got)
		}
	}
}
