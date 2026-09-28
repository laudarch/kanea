package main

import (
	"log/slog"
	"strings"
	"testing"

	"github.com/m18h/kanea/internal/nodeconfig"
)

// The image GC's wiring guards (§5.2.4, v1.111), in the source-reading
// pattern of registry_wiring_test.go: the defect lives in composite
// literals. A collector missing its policy field would happily sweep an
// air-gapped node's preloaded images; one missing its knobs would accept
// `images { gc { ... } }`, log it as loaded, and quietly run the defaults.

func TestTheAgentWiresEveryImageGCDependency(t *testing.T) {
	required := map[string]string{
		"Store":          "the in-use set is empty and every image reads as garbage",
		"Images":         "New refuses to start",
		"Logger":         "removals and refusals happen silently",
		"NodePullPolicy": "a never-policy node sweeps its preloaded images, unrecoverably",
		"Disabled":       "--image-gc off and images.gc.enabled=false are accepted, logged, and ignored",
		"Interval":       "the gc block's interval is accepted, logged, and ignored",
		"MinAge":         "the gc block's min_age is accepted, logged, and ignored",
		"Keep":           "the gc block's keep is accepted, logged, and ignored",
	}
	found, seen := literalFields(t, "agent.go", "imagegc", "Config")
	if !seen {
		t.Fatal("no imagegc.Config literal in agent.go; this test can no longer see what it guards")
	}
	for field, consequence := range required {
		if !found[field] {
			t.Errorf("agent.go builds imagegc.Config without %s: %s", field, consequence)
		}
	}
}

func TestTheAgentHandsTheCollectorToTheAPI(t *testing.T) {
	found, seen := literalFields(t, "agent.go", "api", "ServerConfig")
	if !seen {
		t.Fatal("no api.ServerConfig literal in agent.go; this test can no longer see what it guards")
	}
	if !found["ImageGC"] {
		t.Error("agent.go builds api.ServerConfig without ImageGC: " +
			"GET /v1/images and POST /v1/images/gc answer 503 on every node")
	}
}

// resolveImageGC is v1.51 precedence: the flag wins, the gc block's enabled
// otherwise, and on when neither says anything - the one default in this
// family whose absence is not off.
func TestResolveImageGCPrecedence(t *testing.T) {
	off := &nodeconfig.ImageGCConfig{Disabled: true}
	on := &nodeconfig.ImageGCConfig{}
	cases := []struct {
		name     string
		flag     string
		fromFile *nodeconfig.ImageGCConfig
		disabled bool
	}{
		{"neither says anything means on", "", nil, false},
		{"the block's enabled=false stands", "", off, true},
		{"a block without enabled means on", "", on, false},
		{"--image-gc off wins over the block", "off", on, true},
		{"--image-gc on wins over the block", "on", off, false},
		{"--image-gc off with no file", "off", nil, true},
	}
	logger := slog.New(slog.DiscardHandler)
	for _, tc := range cases {
		disabled, err := resolveImageGC(tc.flag, tc.fromFile, "/etc/kanea/kanea.hcl", logger)
		if err != nil {
			t.Errorf("%s: %v", tc.name, err)
			continue
		}
		if disabled != tc.disabled {
			t.Errorf("%s: disabled = %v, want %v", tc.name, disabled, tc.disabled)
		}
	}
}

func TestResolveImageGCRefusesAnUnknownMode(t *testing.T) {
	_, err := resolveImageGC("sometimes", nil, "", slog.New(slog.DiscardHandler))
	if err == nil || !strings.Contains(err.Error(), "sometimes") {
		t.Fatalf("resolveImageGC(\"sometimes\") = %v; want a refusal naming it", err)
	}
}
