package yamux

import (
	"testing"
)

func TestConst(t *testing.T) {
	if protoVersion != 0 {
		t.Fatalf("bad: %v", protoVersion)
	}

	if typeData != 0 {
		t.Fatalf("bad: %v", typeData)
	}
	if typeWindowUpdate != 1 {
		t.Fatalf("bad: %v", typeWindowUpdate)
	}
	if typePing != 2 {
		t.Fatalf("bad: %v", typePing)
	}
	if typeGoAway != 3 {
		t.Fatalf("bad: %v", typeGoAway)
	}

	if flagSYN != 1 {
		t.Fatalf("bad: %v", flagSYN)
	}
	if flagACK != 2 {
		t.Fatalf("bad: %v", flagACK)
	}
	if flagFIN != 4 {
		t.Fatalf("bad: %v", flagFIN)
	}
	if flagRST != 8 {
		t.Fatalf("bad: %v", flagRST)
	}

	if goAwayNormal != 0 {
		t.Fatalf("bad: %v", goAwayNormal)
	}
	if goAwayProtoErr != 1 {
		t.Fatalf("bad: %v", goAwayProtoErr)
	}
	if goAwayInternalErr != 2 {
		t.Fatalf("bad: %v", goAwayInternalErr)
	}

	if headerSize != 12 {
		t.Fatalf("bad header size")
	}
}

func TestEncodeDecode(t *testing.T) {
	hdr := header(make([]byte, headerSize))
	hdr.encode(typeWindowUpdate, flagACK|flagRST, 1234, 4321)

	if hdr.Version() != protoVersion {
		t.Fatalf("bad: %v", hdr)
	}
	if hdr.MsgType() != typeWindowUpdate {
		t.Fatalf("bad: %v", hdr)
	}
	if hdr.Flags() != flagACK|flagRST {
		t.Fatalf("bad: %v", hdr)
	}
	if hdr.StreamID() != 1234 {
		t.Fatalf("bad: %v", hdr)
	}
	if hdr.Length() != 4321 {
		t.Fatalf("bad: %v", hdr)
	}
}
