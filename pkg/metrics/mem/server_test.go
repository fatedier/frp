package mem

import "testing"

func TestAutoTransportMetrics(t *testing.T) {
	m := newServerMetrics()

	m.AutoNegotiation(true)
	m.AutoNegotiation(false)
	m.AutoTransportSelected("quic")
	m.AutoTransportSelected("quic")
	m.AutoTransportClientOnline("quic")
	m.AutoTransportClientOnline("tcp")
	m.AutoTransportClientOffline("tcp")
	m.AutoTransportSwitch("quic", "tcp")
	m.AutoTransportSwitch("tcp", "tcp")
	m.AutoTransportRejected("kcp")

	stats := m.GetServer()
	if stats.AutoNegotiationSuccess != 1 {
		t.Fatalf("expected one successful negotiation, got %d", stats.AutoNegotiationSuccess)
	}
	if stats.AutoNegotiationFailure != 1 {
		t.Fatalf("expected one failed negotiation, got %d", stats.AutoNegotiationFailure)
	}
	if stats.AutoTransportSelections["quic"] != 2 {
		t.Fatalf("expected two quic selections, got %d", stats.AutoTransportSelections["quic"])
	}
	if stats.AutoTransportClientCounts["quic"] != 1 {
		t.Fatalf("expected one online quic client, got %d", stats.AutoTransportClientCounts["quic"])
	}
	if stats.AutoTransportClientCounts["tcp"] != 0 {
		t.Fatalf("expected zero online tcp clients, got %d", stats.AutoTransportClientCounts["tcp"])
	}
	if stats.AutoTransportSwitchCounts["quic->tcp"] != 1 {
		t.Fatalf("expected one quic to tcp switch, got %d", stats.AutoTransportSwitchCounts["quic->tcp"])
	}
	if stats.AutoTransportSwitchCounts["tcp->tcp"] != 0 {
		t.Fatalf("expected same-protocol switches to be ignored, got %d", stats.AutoTransportSwitchCounts["tcp->tcp"])
	}
	if stats.AutoTransportIllegalSelections["kcp"] != 1 {
		t.Fatalf("expected one kcp reject, got %d", stats.AutoTransportIllegalSelections["kcp"])
	}
}
