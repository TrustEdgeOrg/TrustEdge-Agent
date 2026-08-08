package apps

import (
	"bufio"
	"os"
	"strings"
	"unicode"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

// ExecutableKind classifies a resolved CLI file.
type ExecutableKind string

const (
	ExecutableKindMachO   ExecutableKind = "macho"
	ExecutableKindScript  ExecutableKind = "script"
	ExecutableKindUnknown ExecutableKind = "unknown"
)

var openFileFn = os.Open

// CatalogExecutableNames returns exact executable basenames from catalog
// products that are discovered via bounded bin roots (CLI agents and local
// model runtimes).
func CatalogExecutableNames(catalog *identity.Catalog) map[string]struct{} {
	out := make(map[string]struct{})
	if catalog == nil {
		catalog = identity.DefaultCatalog()
	}
	for _, p := range catalog.Products() {
		switch p.Category {
		case identity.ProductCategoryCLIAgent, identity.ProductCategoryLocalModelRuntime:
		default:
			continue
		}
		for _, n := range p.ExecutableNames {
			if n != "" {
				out[n] = struct{}{}
			}
		}
		for _, n := range p.CandidateNames {
			if n != "" {
				out[n] = struct{}{}
			}
		}
	}
	return out
}

// CatalogCLINames is retained for callers; same as CatalogExecutableNames.
func CatalogCLINames(catalog *identity.Catalog) map[string]struct{} {
	return CatalogExecutableNames(catalog)
}

// DetectExecutableKind inspects file magic / shebang without full parsing.
func DetectExecutableKind(path string) (ExecutableKind, string) {
	f, err := openFileFn(path)
	if err != nil {
		return ExecutableKindUnknown, ""
	}
	defer f.Close()

	var hdr [4]byte
	n, _ := f.Read(hdr[:])
	if n >= 4 {
		// Mach-O 32/64 and fat magic (big/little endian).
		magic := uint32(hdr[0])<<24 | uint32(hdr[1])<<16 | uint32(hdr[2])<<8 | uint32(hdr[3])
		switch magic {
		case 0xFEEDFACE, 0xCEFAEDFE, 0xFEEDFACF, 0xCFFAEDFE, 0xCAFEBABE, 0xBEBAFECA:
			return ExecutableKindMachO, ""
		}
	}
	if n >= 2 && hdr[0] == '#' && hdr[1] == '!' {
		_, _ = f.Seek(0, 0)
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			line = strings.TrimPrefix(line, "#!")
			line = strings.TrimSpace(line)
			interp := firstToken(line)
			if strings.HasSuffix(interp, "/env") {
				// #!/usr/bin/env node → next token
				rest := strings.TrimSpace(strings.TrimPrefix(line, interp))
				interp = firstToken(rest)
			}
			return ExecutableKindScript, posixBase(interp)
		}
		return ExecutableKindScript, ""
	}
	return ExecutableKindUnknown, ""
}

func firstToken(s string) string {
	s = strings.TrimSpace(s)
	for i, r := range s {
		if unicode.IsSpace(r) {
			return s[:i]
		}
	}
	return s
}
