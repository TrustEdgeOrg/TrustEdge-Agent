package apps

// HostIDEIdentity is a known IDE installation used as the extension host.
type HostIDEIdentity struct {
	ProductID string // catalog product id: cursor, vscode
	Name      string
	Path      string // .app bundle path
	Family    string // vscode_compatible
}

// ExtensionInstallObservation is a bounded on-disk extension folder observation
// before catalog matching. It is not product identity by itself.
type ExtensionInstallObservation struct {
	Host           HostIDEIdentity
	InstallPath    string
	Profile        string // empty = default profile
	FolderName     string
	PackageManager string // vscode_extension | cursor_extension
}

// ExtensionProvider discovers extensions for a supported IDE family.
// Implementations must not recursively scan the home directory.
type ExtensionProvider interface {
	Supports(host HostIDEIdentity) bool
	DiscoverInstallations(host HostIDEIdentity) ([]ExtensionInstallObservation, error)
}

// defaultExtensionProviders is the ordered list used by the engine.
func defaultExtensionProviders() []ExtensionProvider {
	return []ExtensionProvider{
		&VSCodeCompatibleExtensionProvider{},
	}
}
