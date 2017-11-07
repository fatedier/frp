package socks5

import (
	"testing"
)

func TestStaticCredentials(t *testing.T) {
	creds := StaticCredentials{
		"foo": "bar",
		"baz": "",
	}

	if !creds.Valid("foo", "bar") {
		t.Fatalf("expect valid")
	}

	if !creds.Valid("baz", "") {
		t.Fatalf("expect valid")
	}

	if creds.Valid("foo", "") {
		t.Fatalf("expect invalid")
	}
}
