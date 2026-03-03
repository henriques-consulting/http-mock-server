package handler

import (
	"fmt"
	"http-mock-server/internal/config"
)

func (h *MockHandler) generateRandomBody(spec *config.RandomBodySpec) ([]byte, error) {
	switch spec.Type {
	case "plaintext":
		return h.generatePlaintext(spec.SizeBytes), nil
	case "json":
		return h.generateJSON(spec.SizeBytes)
	case "xml":
		return h.generateXML(spec.SizeBytes), nil
	default:
		return nil, fmt.Errorf("unsupported random body type: %s", spec.Type)
	}
}

// fillAlpha fills buf with random lowercase ASCII letters using batched rand.Read.
// Note: b%26 has slight modulo bias (256 is not divisible by 26), which is acceptable
// for mock server payloads where uniform distribution is not required.
func (h *MockHandler) fillAlpha(buf []byte) {
	h.rand.Read(buf) //nolint:staticcheck // math/rand Read is fine for non-crypto use
	for i, b := range buf {
		buf[i] = 'a' + b%26
	}
}

func (h *MockHandler) generatePlaintext(size int) []byte {
	if size == 0 {
		return []byte{}
	}
	buf := make([]byte, size)
	h.fillAlpha(buf)
	return buf
}

func (h *MockHandler) generateJSON(size int) ([]byte, error) {
	// Wrap random content in a minimal JSON string value: {"":"XXXXX..."}
	// Overhead: {"":""} = 7 bytes
	const prefix = `{"":"`
	const suffix = `"}`
	const overhead = len(prefix) + len(suffix)

	if size < overhead {
		return []byte("{}"), nil
	}

	buf := make([]byte, size)
	copy(buf, prefix)
	h.fillAlpha(buf[len(prefix) : size-len(suffix)])
	copy(buf[size-len(suffix):], suffix)
	return buf, nil
}

func (h *MockHandler) generateXML(size int) []byte {
	// Sizes 7-12:  <r>XXXXX</r>      (overhead 7)
	// Sizes 13+:   <root>XXXXX</root> (overhead 13)
	const shortPrefix = "<r>"
	const shortSuffix = "</r>"
	const longPrefix = "<root>"
	const longSuffix = "</root>"
	const shortOverhead = len(shortPrefix) + len(shortSuffix)
	const longOverhead = len(longPrefix) + len(longSuffix)

	if size < shortOverhead {
		return []byte("<r/>")[:min(size, 4)]
	}

	var prefix, suffix string
	if size < longOverhead {
		prefix, suffix = shortPrefix, shortSuffix
	} else {
		prefix, suffix = longPrefix, longSuffix
	}

	buf := make([]byte, size)
	copy(buf, prefix)
	h.fillAlpha(buf[len(prefix) : size-len(suffix)])
	copy(buf[size-len(suffix):], suffix)
	return buf
}
