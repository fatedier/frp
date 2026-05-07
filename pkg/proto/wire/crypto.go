// Copyright 2026 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wire

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"runtime"

	"golang.org/x/sys/cpu"
)

const (
	AEADAlgorithmAES256GCM         = "aes-256-gcm"
	AEADAlgorithmXChaCha20Poly1305 = "xchacha20-poly1305"

	CryptoRandomSize = 32

	cryptoTranscriptLabel = "frp wire v2 crypto transcript"
)

var supportedAEADAlgorithms = []string{
	AEADAlgorithmAES256GCM,
	AEADAlgorithmXChaCha20Poly1305,
}

type CryptoContext struct {
	Algorithm      string
	TranscriptHash []byte
}

func NewClientHello(bootstrap BootstrapInfo) (ClientHello, error) {
	clientRandom, err := newCryptoRandom()
	if err != nil {
		return ClientHello{}, err
	}
	return clientHelloWithCryptoRandom(bootstrap, clientRandom), nil
}

func NewServerHello(clientHello ClientHello) (ServerHello, error) {
	if err := ValidateClientHello(clientHello); err != nil {
		return ServerHello{}, err
	}
	algorithm, ok := SelectAEADAlgorithm(clientHello.Capabilities.Crypto.Algorithms)
	if !ok {
		return ServerHello{}, fmt.Errorf("no supported crypto algorithm")
	}
	serverRandom, err := newCryptoRandom()
	if err != nil {
		return ServerHello{}, err
	}
	return ServerHello{
		Selected: ServerSelection{
			Message: MessageSelection{
				Codec: MessageCodecJSON,
			},
			Crypto: CryptoSelection{
				Algorithm:    algorithm,
				ServerRandom: serverRandom,
			},
		},
	}, nil
}

func ValidateCryptoCapabilities(c CryptoCapabilities) error {
	if len(c.ClientRandom) != CryptoRandomSize {
		return fmt.Errorf("invalid crypto client random length %d, want %d", len(c.ClientRandom), CryptoRandomSize)
	}
	if _, ok := SelectAEADAlgorithm(c.Algorithms); !ok {
		return fmt.Errorf("no supported crypto algorithm")
	}
	return nil
}

func ValidateServerHelloForClient(clientHello ClientHello, serverHello ServerHello) error {
	if serverHello.Selected.Message.Codec != MessageCodecJSON {
		return fmt.Errorf("unsupported selected message codec: %s", serverHello.Selected.Message.Codec)
	}
	cryptoSelection := serverHello.Selected.Crypto
	if !IsSupportedAEADAlgorithm(cryptoSelection.Algorithm) {
		return fmt.Errorf("unknown selected crypto algorithm: %s", cryptoSelection.Algorithm)
	}
	if !Supports(clientHello.Capabilities.Crypto.Algorithms, cryptoSelection.Algorithm) {
		return fmt.Errorf("selected crypto algorithm was not advertised by client: %s", cryptoSelection.Algorithm)
	}
	if len(cryptoSelection.ServerRandom) != CryptoRandomSize {
		return fmt.Errorf("invalid crypto server random length %d, want %d", len(cryptoSelection.ServerRandom), CryptoRandomSize)
	}
	return nil
}

func NewCryptoContext(algorithm string, clientHelloPayload, serverHelloPayload []byte) *CryptoContext {
	return &CryptoContext{
		Algorithm:      algorithm,
		TranscriptHash: HashCryptoTranscript(clientHelloPayload, serverHelloPayload),
	}
}

func NewClientCryptoContext(clientHelloPayload, serverHelloPayload []byte) (*CryptoContext, error) {
	var clientHello ClientHello
	if err := json.Unmarshal(clientHelloPayload, &clientHello); err != nil {
		return nil, fmt.Errorf("decode ClientHello transcript: %w", err)
	}
	var serverHello ServerHello
	if err := json.Unmarshal(serverHelloPayload, &serverHello); err != nil {
		return nil, fmt.Errorf("decode ServerHello transcript: %w", err)
	}
	if err := ValidateServerHelloForClient(clientHello, serverHello); err != nil {
		return nil, err
	}

	return NewCryptoContext(serverHello.Selected.Crypto.Algorithm, clientHelloPayload, serverHelloPayload), nil
}

func HashCryptoTranscript(clientHelloPayload, serverHelloPayload []byte) []byte {
	h := sha256.New()
	_, _ = h.Write([]byte(cryptoTranscriptLabel))
	writeCryptoTranscriptPart(h, "client hello", clientHelloPayload)
	writeCryptoTranscriptPart(h, "server hello", serverHelloPayload)
	return h.Sum(nil)
}

func writeCryptoTranscriptPart(h hash.Hash, label string, payload []byte) {
	var length [8]byte
	binary.BigEndian.PutUint64(length[:], uint64(len(payload)))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write([]byte(label))
	_, _ = h.Write([]byte{0})
	_, _ = h.Write(length[:])
	_, _ = h.Write(payload)
}

func PreferredAEADAlgorithms() []string {
	if hasFastAESGCM() {
		return []string{AEADAlgorithmAES256GCM, AEADAlgorithmXChaCha20Poly1305}
	}
	return []string{AEADAlgorithmXChaCha20Poly1305, AEADAlgorithmAES256GCM}
}

func SelectAEADAlgorithm(clientAlgorithms []string) (string, bool) {
	for _, algorithm := range clientAlgorithms {
		if IsSupportedAEADAlgorithm(algorithm) {
			return algorithm, true
		}
	}
	return "", false
}

func IsSupportedAEADAlgorithm(algorithm string) bool {
	return Supports(supportedAEADAlgorithms, algorithm)
}

func newCryptoRandom() ([]byte, error) {
	b := make([]byte, CryptoRandomSize)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate crypto random: %w", err)
	}
	return b, nil
}

func hasFastAESGCM() bool {
	switch runtime.GOARCH {
	case "amd64":
		return cpu.X86.HasAES &&
			cpu.X86.HasPCLMULQDQ &&
			cpu.X86.HasSSE41 &&
			cpu.X86.HasSSSE3
	case "arm64":
		return cpu.ARM64.HasAES && cpu.ARM64.HasPMULL
	case "s390x":
		return cpu.S390X.HasAES &&
			cpu.S390X.HasAESCTR &&
			cpu.S390X.HasGHASH
	case "ppc64", "ppc64le":
		// Go's ppc64/ppc64le port targets POWER8+, which has AES instructions;
		// x/sys/cpu does not expose a PPC64 AES feature flag.
		return true
	default:
		return false
	}
}
