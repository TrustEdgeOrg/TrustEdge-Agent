package apps

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

const fingerprintTimeout = 800 * time.Millisecond
const fingerprintMaxBody = 64 << 10 // 64 KiB

// RuntimeFingerprint is bounded metadata from a local inference server.
type RuntimeFingerprint struct {
	ProductID string
	Version   string
	Models    int
	OK        bool
}

// RuntimeFingerprintProvider probes a known product's local metadata API.
// Must never send prompts or trigger inference.
type RuntimeFingerprintProvider interface {
	Supports(productID string) bool
	Probe(ctx context.Context, baseURL string) (RuntimeFingerprint, error)
}

// OllamaFingerprintProvider GETs Ollama /api/version and /api/tags.
type OllamaFingerprintProvider struct {
	Client *http.Client
}

func (p OllamaFingerprintProvider) Supports(productID string) bool {
	return productID == identity.ProductOllamaID
}

func (p OllamaFingerprintProvider) Probe(ctx context.Context, baseURL string) (RuntimeFingerprint, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return RuntimeFingerprint{}, fmt.Errorf("empty base URL")
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: fingerprintTimeout}
	}
	out := RuntimeFingerprint{ProductID: identity.ProductOllamaID}

	verBody, err := httpGetLimited(ctx, client, baseURL+"/api/version")
	if err != nil {
		return out, err
	}
	var ver struct {
		Version string `json:"version"`
	}
	_ = json.Unmarshal(verBody, &ver)
	out.Version = strings.TrimSpace(ver.Version)

	tagsBody, err := httpGetLimited(ctx, client, baseURL+"/api/tags")
	if err == nil {
		var tags struct {
			Models []json.RawMessage `json:"models"`
		}
		if json.Unmarshal(tagsBody, &tags) == nil {
			out.Models = len(tags.Models)
		}
	}
	out.OK = out.Version != "" || out.Models > 0
	return out, nil
}

func httpGetLimited(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, fingerprintMaxBody))
}

func loopbackBaseURL(listeners []ListenerInfo) string {
	for _, l := range listeners {
		if !isLoopbackAddr(l.Addr) && l.Addr != "0.0.0.0" && l.Addr != "::" {
			continue
		}
		host := l.Addr
		if host == "0.0.0.0" || host == "::" || host == "" {
			host = "127.0.0.1"
		}
		if host == "::1" {
			host = "[::1]"
		}
		if l.Port <= 0 {
			continue
		}
		return fmt.Sprintf("http://%s:%d", host, l.Port)
	}
	return ""
}

func (e *Engine) applyRuntimeFingerprint(entry *InventoryEntry) {
	if e == nil || entry == nil || !entry.Serving || !isLocalModelRuntime(entry) {
		return
	}
	p := entry.Identification.Product
	if p == nil {
		return
	}
	base := loopbackBaseURL(entry.Listeners)
	if base == "" {
		return
	}
	provider := RuntimeFingerprintProvider(OllamaFingerprintProvider{})
	if !provider.Supports(p.ID) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), fingerprintTimeout)
	defer cancel()
	fp, err := provider.Probe(ctx, base)
	if err != nil || !fp.OK {
		return
	}
	if fp.Version != "" {
		entry.RuntimeVersion = fp.Version
		if entry.Identity.Version == "" {
			entry.Identity.Version = fp.Version
		}
	}
	if fp.Models > 0 {
		entry.ModelsAvailable = fp.Models
	}
	entry.ModelActiveUnknown = true
	if !hasEvidence(entry.Identification.Matched, identity.EvidenceRuntimeFingerprint) {
		entry.Identification.Matched = append(entry.Identification.Matched, identity.EvidenceRuntimeFingerprint)
	}
}
