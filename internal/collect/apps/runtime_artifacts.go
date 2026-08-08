package apps

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

const (
	ggufMagic       = "GGUF"
	ggufHeaderLimit = 16
	maxArtifactWalk = 64 // max directory entries inspected
)

// ArtifactScanResult is bounded model/runtime metadata (never file contents).
type ArtifactScanResult struct {
	ModelsAvailable int
	ModelFormat     string
	Found           bool
}

// ScanRuntimeArtifacts inspects only catalog ArtifactPathHints for an identified
// runtime. No recursive home walk; no multi-GB hashing.
func ScanRuntimeArtifacts(product *identity.KnownAIProduct, home string) ArtifactScanResult {
	var out ArtifactScanResult
	if product == nil || len(product.ArtifactPathHints) == 0 {
		return out
	}
	for _, hint := range product.ArtifactPathHints {
		root := expandArtifactHint(hint, home)
		if root == "" {
			continue
		}
		fi, err := os.Stat(root)
		if err != nil || !fi.IsDir() {
			continue
		}
		out.Found = true
		n, format := countModelArtifacts(root)
		out.ModelsAvailable += n
		if format != "" && out.ModelFormat == "" {
			out.ModelFormat = format
		}
	}
	return out
}

func expandArtifactHint(hint, home string) string {
	hint = strings.TrimSpace(hint)
	if hint == "" {
		return ""
	}
	if strings.HasPrefix(hint, "~/") {
		if home == "" {
			h, err := homeDirFn()
			if err != nil || h == "" {
				return ""
			}
			home = h
		}
		return filepath.Join(home, hint[2:])
	}
	return hint
}

func countModelArtifacts(root string) (int, string) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, ""
	}
	count := 0
	format := ""
	n := 0
	for _, e := range entries {
		n++
		if n > maxArtifactWalk {
			break
		}
		name := e.Name()
		full := filepath.Join(root, name)
		if e.IsDir() {
			// Ollama stores models under manifests/blobs — count one level of children.
			sub, err := os.ReadDir(full)
			if err != nil {
				continue
			}
			if len(sub) > 0 && (name == "models" || name == "manifests" || name == "blobs") {
				count += minInt(len(sub), maxArtifactWalk)
			}
			continue
		}
		lower := strings.ToLower(name)
		if strings.HasSuffix(lower, ".gguf") {
			count++
			if format == "" && ggufHeaderOK(full) {
				format = "GGUF"
			} else if format == "" {
				format = "GGUF"
			}
		}
	}
	return count, format
}

func ggufHeaderOK(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, ggufHeaderLimit)
	n, _ := f.Read(buf)
	if n < 4 {
		return false
	}
	return string(buf[:4]) == ggufMagic
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (e *Engine) applyRuntimeArtifacts(entry *InventoryEntry) {
	if e == nil || entry == nil || !isLocalModelRuntime(entry) {
		return
	}
	p := entry.Identification.Product
	if p == nil || len(p.ArtifactPathHints) == 0 {
		return
	}
	res := ScanRuntimeArtifacts(p, "")
	if !res.Found {
		return
	}
	if res.ModelsAvailable > entry.ModelsAvailable {
		entry.ModelsAvailable = res.ModelsAvailable
	}
	if res.ModelFormat != "" {
		entry.ModelFormat = res.ModelFormat
	}
	entry.ModelActiveUnknown = true
	if !hasEvidence(entry.Identification.Matched, identity.EvidenceModelArtifact) {
		entry.Identification.Matched = append(entry.Identification.Matched, identity.EvidenceModelArtifact)
	}
}
