package identity

// Catalog is a versionable set of known AI products.
type Catalog struct {
	products []KnownAIProduct
}

// DefaultCatalog returns the built-in known-AI product catalog.
func DefaultCatalog() *Catalog {
	return NewCatalog(builtinProducts()...)
}

// NewCatalog constructs a catalog from the given products.
func NewCatalog(products ...KnownAIProduct) *Catalog {
	cp := make([]KnownAIProduct, len(products))
	copy(cp, products)
	return &Catalog{products: cp}
}

// Products returns a copy of catalog entries.
func (c *Catalog) Products() []KnownAIProduct {
	if c == nil {
		return nil
	}
	out := make([]KnownAIProduct, len(c.products))
	copy(out, c.products)
	return out
}

// Lookup returns the product with the given stable ID, if present.
func (c *Catalog) Lookup(id string) (KnownAIProduct, bool) {
	if c == nil {
		return KnownAIProduct{}, false
	}
	for _, p := range c.products {
		if p.ID == id {
			return p, true
		}
	}
	return KnownAIProduct{}, false
}
