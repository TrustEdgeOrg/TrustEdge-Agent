package apps

import (
	"context"
	"testing"
	"time"

	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/collect/process"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/constants"
	"github.com/TrustEdgeOrg/TrustEdge-Agent/internal/identity"
)

func TestEngineCLIRunningFromResolvedPath(t *testing.T) {
	resolved := "/opt/homebrew/Cellar/claude-code/1.0/bin/claude"
	inv := "/opt/homebrew/bin/claude"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path:           resolved,
			Executable:     "claude",
			ExecutablePath: resolved,
			InvocationPath: inv,
			ResolvedPath:   resolved,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{
				PID:               99,
				Executable:        resolved,
				Comm:              "claude",
				StartTimeUnixNano: 1000,
			}}, nil
		},
	})
	invList, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(invList) != 1 || !invList[0].Running || !invList[0].Installed {
		t.Fatalf("%+v", invList)
	}
	if invList[0].Identification.Product == nil || invList[0].Identification.Product.ID != identity.ProductClaudeCodeID {
		t.Fatalf("product=%v", invList[0].Identification.Product)
	}
}

func TestEngineCLIBasenameMapsViaInstallIndex(t *testing.T) {
	resolved := "/Users/x/.local/bin/codex-real"
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path:           resolved,
			Executable:     "codex",
			ExecutablePath: resolved,
			InvocationPath: "/Users/x/.local/bin/codex",
			ResolvedPath:   resolved,
		}}},
		Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{
				PID:        7,
				Executable: "codex", // basename-only (ps-style)
				Comm:       "codex",
			}}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !got[0].Running {
		t.Fatalf("want installed+running via basename map, got %+v", got)
	}
}

func TestEngineCLIExitClearsRunning(t *testing.T) {
	resolved := "/opt/homebrew/bin/gemini"
	list := []process.ProcessInfo{{PID: 5, Executable: resolved, Comm: "gemini", StartTimeUnixNano: 1}}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: resolved, Executable: "gemini", ExecutablePath: resolved, ResolvedPath: resolved,
		}}},
		Signer:    nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return list, nil },
	})
	first, _ := eng.Inventory()
	if !first[0].Running {
		t.Fatal("expected running")
	}
	list = nil
	eng.NoteExit(5)
	second, _ := eng.Inventory()
	if second[0].Running {
		t.Fatal("EXIT/empty snapshot must clear running")
	}
}

func TestEngineCLIPIDReuseDifferentStartTime(t *testing.T) {
	resolved := "/opt/homebrew/bin/opencode"
	list := []process.ProcessInfo{{PID: 42, Executable: resolved, Comm: "opencode", StartTimeUnixNano: 10}}
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{apps: []identity.ApplicationIdentity{{
			Path: resolved, Executable: "opencode", ExecutablePath: resolved, ResolvedPath: resolved,
		}}},
		Signer:    nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return list, nil },
	})
	_, _ = eng.Inventory()
	// Same PID, different process (start time) that is not the CLI.
	list = []process.ProcessInfo{{PID: 42, Executable: "/usr/bin/true", Comm: "true", StartTimeUnixNano: 99}}
	got, _ := eng.Inventory()
	if got[0].Running {
		t.Fatal("PID reuse must not keep CLI running")
	}
}

func TestEngineGenericNodeNotMatched(t *testing.T) {
	eng := NewEngine(EngineConfig{
		Discoverer: stubDiscoverer{},
		Signer:     nil,
		ListProcs: func() ([]process.ProcessInfo, error) {
			return []process.ProcessInfo{{PID: 1, Executable: "/usr/local/bin/node", Comm: "node"}}, nil
		},
	})
	got, err := eng.Inventory()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("node alone must not appear: %+v", got)
	}
}

func TestRuntimeFeedWakesOnCLIAndExitTracked(t *testing.T) {
	feed := NewRuntimeFeed(nil, identity.NewMatcher(identity.DefaultCatalog()))
	eng := NewEngine(EngineConfig{Discoverer: stubDiscoverer{}, Signer: nil,
		ListProcs: func() ([]process.ProcessInfo, error) { return nil, nil },
	})
	feed.SetEngine(eng)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go feed.Run(ctx)

	feed.ObserveChange(collect.Change{
		Type: constants.TypeProcessStart,
		Payload: map[string]any{
			"pid": 11, "comm": "claude", "executable": "/opt/homebrew/bin/claude",
			"start_time_unix_nano": int64(123),
		},
	})
	select {
	case <-feed.Wakes():
	case <-time.After(2 * time.Second):
		t.Fatal("expected wake for claude CLI")
	}

	feed.ObserveChange(collect.Change{
		Type:    constants.TypeProcessExit,
		Payload: map[string]any{"pid": 11, "executable": "/usr/bin/true", "comm": "true"},
	})
	select {
	case <-feed.Wakes():
	case <-time.After(2 * time.Second):
		t.Fatal("expected wake on tracked PID exit even if exit path is unrelated")
	}
}
