package identity

// Local model runtime catalog entries.
//
// Category is always ProductCategoryLocalModelRuntime — never cli_agent or
// AI_AGENT. PackageManagers, PackageIdentifiers, SigningIdentifiers, TeamIDs,
// DefaultLocalEndpoints, and ArtifactPathHints are intentionally empty
// (unresolved). Do not invent Homebrew formulas, Team IDs, ports, or
// artifact paths until verified from an authentic install or official
// metadata checked into this repo.
//
// ExecutableNames are discovery hints only; name alone never VERIFIED.

func builtinRuntimeProducts() []KnownAIProduct {
	return []KnownAIProduct{
		ollamaRuntimeProduct(),
		llamaCppRuntimeProduct(),
	}
}

func ollamaRuntimeProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductOllamaID,
		Name:            "Ollama",
		Vendor:          "Ollama",
		Category:        ProductCategoryLocalModelRuntime,
		ExecutableNames: []string{"ollama"},
		// Artifact path from Ollama public docs (models stored under ~/.ollama).
		// https://github.com/ollama/ollama/blob/main/docs/faq.md
		ArtifactPathHints: []string{"~/.ollama"},
		// Package/signing/endpoints unresolved — not invented.
	}
}

func llamaCppRuntimeProduct() KnownAIProduct {
	return KnownAIProduct{
		ID:              ProductLlamaCppID,
		Name:            "llama.cpp",
		Vendor:          "ggerganov",
		Category:        ProductCategoryLocalModelRuntime,
		ExecutableNames: []string{"llama-server", "llama-cli"},
		RuntimeFamily:   RuntimeFamilyLlamaCppCompatible,
		// Self-built / renamed binaries are common; exact VERIFIED identity
		// requires package provenance or stronger evidence.
	}
}

const (
	ProductOllamaID   = "ollama"
	ProductLlamaCppID = "llama_cpp"

	RuntimeFamilyLlamaCppCompatible = "llama_cpp_compatible"
)
