package identity

import "testing"

func TestCachePutGetAndLRUEviction(t *testing.T) {
	c := NewCache(2)
	k1 := FileFingerprint("/a.app", 1, 10)
	k2 := FileFingerprint("/b.app", 1, 10)
	k3 := FileFingerprint("/c.app", 1, 10)

	c.PutIdentity(k1, ApplicationIdentity{Path: "/a.app", BundleID: "a"})
	c.PutIdentity(k2, ApplicationIdentity{Path: "/b.app", BundleID: "b"})
	if c.Len() != 2 {
		t.Fatalf("Len=%d", c.Len())
	}

	// Touch k1 so k2 is oldest.
	if _, ok := c.Get(k1); !ok {
		t.Fatal("k1 missing")
	}
	c.PutIdentity(k3, ApplicationIdentity{Path: "/c.app", BundleID: "c"})
	if c.Len() != 2 {
		t.Fatalf("Len=%d after eviction", c.Len())
	}
	if _, ok := c.Get(k2); ok {
		t.Fatal("k2 should have been evicted")
	}
	if e, ok := c.Get(k1); !ok || e.Identity.BundleID != "a" {
		t.Fatalf("k1=%v ok=%v", e, ok)
	}
	if e, ok := c.Get(k3); !ok || e.Identity.BundleID != "c" {
		t.Fatalf("k3=%v ok=%v", e, ok)
	}
}

func TestCacheInvalidateOnChange(t *testing.T) {
	c := NewCache(8)
	kOld := FileFingerprint("/Applications/Cursor.app", 100, 50)
	kNew := FileFingerprint("/Applications/Cursor.app", 200, 50)
	c.PutIdentity(kOld, ApplicationIdentity{Version: "1"})
	c.PutIdentification(kOld, IdentificationResult{Confidence: ConfidenceLow})

	if _, ok := c.Get(kOld); !ok {
		t.Fatal("expected old key")
	}
	if _, ok := c.Get(kNew); ok {
		t.Fatal("new fingerprint must miss until populated")
	}

	c.InvalidatePath("/Applications/Cursor.app")
	if _, ok := c.Get(kOld); ok {
		t.Fatal("expected invalidation")
	}
	if c.Len() != 0 {
		t.Fatalf("Len=%d", c.Len())
	}
}

func TestCacheIdentificationMerge(t *testing.T) {
	c := NewCache(4)
	k := FileFingerprint("/x.app", 1, 1)
	c.PutIdentity(k, ApplicationIdentity{Path: "/x.app"})
	c.PutIdentification(k, IdentificationResult{Confidence: ConfidenceHigh})
	e, ok := c.Get(k)
	if !ok || !e.HasIdentity || !e.HasIdentification {
		t.Fatalf("entry=%+v ok=%v", e, ok)
	}
	if e.Identification.Confidence != ConfidenceHigh {
		t.Fatalf("Confidence=%s", e.Identification.Confidence)
	}
}

func TestCacheCapacityBounded(t *testing.T) {
	c := NewCache(3)
	for i := 0; i < 20; i++ {
		k := FileFingerprint("/app", int64(i), int64(i))
		c.PutIdentity(k, ApplicationIdentity{Version: "x"})
	}
	if c.Len() > c.Cap() {
		t.Fatalf("Len=%d Cap=%d", c.Len(), c.Cap())
	}
	if c.Len() != 3 {
		t.Fatalf("Len=%d want 3", c.Len())
	}
}
