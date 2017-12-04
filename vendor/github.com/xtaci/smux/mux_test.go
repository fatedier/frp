package smux

import (
	"bytes"
	"testing"
)

type buffer struct {
	bytes.Buffer
}

func (b *buffer) Close() error {
	b.Buffer.Reset()
	return nil
}

func TestConfig(t *testing.T) {
	VerifyConfig(DefaultConfig())

	config := DefaultConfig()
	config.KeepAliveInterval = 0
	err := VerifyConfig(config)
	t.Log(err)
	if err == nil {
		t.Fatal(err)
	}

	config = DefaultConfig()
	config.KeepAliveInterval = 10
	config.KeepAliveTimeout = 5
	err = VerifyConfig(config)
	t.Log(err)
	if err == nil {
		t.Fatal(err)
	}

	config = DefaultConfig()
	config.MaxFrameSize = 0
	err = VerifyConfig(config)
	t.Log(err)
	if err == nil {
		t.Fatal(err)
	}

	config = DefaultConfig()
	config.MaxFrameSize = 65536
	err = VerifyConfig(config)
	t.Log(err)
	if err == nil {
		t.Fatal(err)
	}

	config = DefaultConfig()
	config.MaxReceiveBuffer = 0
	err = VerifyConfig(config)
	t.Log(err)
	if err == nil {
		t.Fatal(err)
	}

	var bts buffer
	if _, err := Server(&bts, config); err == nil {
		t.Fatal("server started with wrong config")
	}

	if _, err := Client(&bts, config); err == nil {
		t.Fatal("client started with wrong config")
	}
}
