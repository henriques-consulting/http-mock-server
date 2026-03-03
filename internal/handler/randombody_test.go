package handler

import (
	"encoding/json"
	"encoding/xml"
	"http-mock-server/internal/config"
	"io"
	"math/rand"
	"strings"
	"sync"
	"testing"
)

func newTestHandler() *MockHandler {
	cfg := &config.Config{}
	r := rand.New(rand.NewSource(42))
	return &MockHandler{
		config:       cfg,
		rand:         r,
		cachedBodies: make(map[*config.RandomBodySpec][]byte),
	}
}

func TestGeneratePlaintext_ExactSize(t *testing.T) {
	h := newTestHandler()

	sizes := []int{0, 1, 10, 100, 512, 1024, 4096}
	for _, size := range sizes {
		data := h.generatePlaintext(size)
		if len(data) != size {
			t.Errorf("generatePlaintext(%d) produced %d bytes, want %d", size, len(data), size)
		}
	}
}

func TestGeneratePlaintext_ValidChars(t *testing.T) {
	h := newTestHandler()

	data := h.generatePlaintext(1000)
	for i, b := range data {
		if b < 'a' || b > 'z' {
			t.Errorf("byte %d: character %q is not a lowercase letter", i, string(b))
		}
	}
}

func TestGeneratePlaintext_ZeroSize(t *testing.T) {
	h := newTestHandler()

	data := h.generatePlaintext(0)
	if len(data) != 0 {
		t.Errorf("expected empty slice, got %d bytes", len(data))
	}
}

func TestGenerateJSON_ValidJSON(t *testing.T) {
	h := newTestHandler()

	sizes := []int{2, 9, 10, 50, 100, 500, 1024, 2048}
	for _, size := range sizes {
		data, err := h.generateJSON(size)
		if err != nil {
			t.Errorf("generateJSON(%d) error: %v", size, err)
			continue
		}

		var obj interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			t.Errorf("generateJSON(%d) produced invalid JSON: %v\ndata: %s", size, err, string(data))
		}
	}
}

func TestGenerateJSON_ExactSize(t *testing.T) {
	h := newTestHandler()

	// Sizes 3-6 fall back to "{}" (2 bytes) and are not exact; size 2 and size >= 7 are exact.
	sizes := []int{2, 7, 9, 10, 11, 12, 15, 20, 50, 100, 500, 1024, 2048}
	for _, size := range sizes {
		data, err := h.generateJSON(size)
		if err != nil {
			t.Errorf("generateJSON(%d) error: %v", size, err)
			continue
		}

		if len(data) != size {
			t.Errorf("generateJSON(%d) produced %d bytes, want %d\ndata: %s", size, len(data), size, string(data))
		}
	}
}

func TestGenerateJSON_MinimalSize(t *testing.T) {
	h := newTestHandler()

	data, err := h.generateJSON(2)
	if err != nil {
		t.Fatalf("generateJSON(2) error: %v", err)
	}
	if string(data) != "{}" {
		t.Errorf("expected {}, got %s", string(data))
	}
}

func TestGenerateXML_ValidXML(t *testing.T) {
	h := newTestHandler()

	sizes := []int{7, 13, 20, 50, 100, 500, 1024}
	for _, size := range sizes {
		data := h.generateXML(size)

		decoder := xml.NewDecoder(strings.NewReader(string(data)))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("generateXML(%d) produced invalid XML: %v\ndata: %s", size, err, string(data))
				break
			}
		}
	}
}

func TestGenerateXML_ExactSize(t *testing.T) {
	h := newTestHandler()

	sizes := []int{7, 13, 20, 50, 100, 500, 1024}
	for _, size := range sizes {
		data := h.generateXML(size)
		if len(data) != size {
			t.Errorf("generateXML(%d) produced %d bytes, want %d\ndata: %s", size, len(data), size, string(data))
		}
	}
}

func TestGenerateXML_ShortRootWrapper(t *testing.T) {
	h := newTestHandler()

	// Sizes 7-12 should use the short <r></r> wrapper (7 bytes overhead)
	for size := 7; size <= 12; size++ {
		data := h.generateXML(size)
		s := string(data)
		if !strings.HasPrefix(s, "<r>") {
			t.Errorf("generateXML(%d) expected short root <r>, got: %s", size, s)
		}
		if !strings.HasSuffix(s, "</r>") {
			t.Errorf("generateXML(%d) expected closing </r>, got: %s", size, s)
		}
		if len(data) != size {
			t.Errorf("generateXML(%d) produced %d bytes", size, len(data))
		}

		// Verify valid XML
		decoder := xml.NewDecoder(strings.NewReader(s))
		for {
			_, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Errorf("generateXML(%d) produced invalid XML: %v\ndata: %s", size, err, s)
				break
			}
		}
	}
}

func TestGenerateXML_HasRootWrapper(t *testing.T) {
	h := newTestHandler()

	data := h.generateXML(100)
	s := string(data)
	if !strings.HasPrefix(s, "<root>") {
		t.Errorf("expected XML to start with <root>, got: %s", s[:20])
	}
	if !strings.HasSuffix(s, "</root>") {
		t.Errorf("expected XML to end with </root>, got: %s", s[len(s)-20:])
	}
}

func TestGenerateRandomBody_Dispatcher(t *testing.T) {
	h := newTestHandler()

	tests := []struct {
		name string
		spec *config.RandomBodySpec
	}{
		{"plaintext", &config.RandomBodySpec{Type: "plaintext", SizeBytes: 256}},
		{"json", &config.RandomBodySpec{Type: "json", SizeBytes: 256}},
		{"xml", &config.RandomBodySpec{Type: "xml", SizeBytes: 256}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := h.generateRandomBody(tt.spec)
			if err != nil {
				t.Fatalf("generateRandomBody(%s) error: %v", tt.name, err)
			}
			if len(data) != tt.spec.SizeBytes {
				t.Errorf("expected %d bytes, got %d", tt.spec.SizeBytes, len(data))
			}
		})
	}
}

func TestGenerateRandomBody_UnsupportedType(t *testing.T) {
	h := newTestHandler()

	_, err := h.generateRandomBody(&config.RandomBodySpec{Type: "html", SizeBytes: 100})
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestCachedBodies_ConcurrentReads(t *testing.T) {
	spec := &config.RandomBodySpec{Type: "plaintext", SizeBytes: 1024}
	cfg := &config.Config{
		Requests: []config.RequestRule{
			{
				Path:   "/random",
				Method: "GET",
				Response: config.ResponseSpec{
					StatusCode: 200,
					RandomBody: spec,
				},
			},
		},
	}

	r := rand.New(rand.NewSource(42))
	h := NewMockHandlerWithRand(cfg, r)

	// Verify the body was pre-generated
	data, ok := h.cachedBodies[spec]
	if !ok {
		t.Fatal("expected cached body to be pre-generated")
	}
	if len(data) != 1024 {
		t.Fatalf("expected 1024 bytes, got %d", len(data))
	}

	// Concurrent reads should be safe (no locking needed for reads)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cached := h.cachedBodies[spec]
			if len(cached) != 1024 {
				t.Errorf("concurrent read got %d bytes, want 1024", len(cached))
			}
		}()
	}
	wg.Wait()
}
