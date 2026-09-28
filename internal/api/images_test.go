package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/m18h/kanea/internal/api"
	"github.com/m18h/kanea/internal/auth"
	"github.com/m18h/kanea/internal/imagegc"
)

// fakeCollector stands in for the daemon's image GC.
type fakeCollector struct {
	images  []imagegc.Image
	summary imagegc.Summary
	refusal string
	sweeps  int
}

func (f *fakeCollector) List(context.Context) ([]imagegc.Image, error) { return f.images, nil }
func (f *fakeCollector) Refusal() string                               { return f.refusal }
func (f *fakeCollector) Once(context.Context) (imagegc.Summary, error) {
	f.sweeps++
	return f.summary, nil
}

func withCollector(f *fakeCollector) func(*api.ServerConfig) {
	return func(cfg *api.ServerConfig) { cfg.ImageGC = f }
}

func TestImagesIs503WithoutACollector(t *testing.T) {
	h := newHarness(t)
	if status, _ := h.raw(t, http.MethodGet, api.PathImages); status != http.StatusServiceUnavailable {
		t.Errorf("images = %d, want 503", status)
	}
	if status, _ := h.raw(t, http.MethodPost, api.PathImagesGC); status != http.StatusServiceUnavailable {
		t.Errorf("images gc = %d, want 503", status)
	}
}

func TestImagesListServesTheCollectorsView(t *testing.T) {
	fake := &fakeCollector{images: []imagegc.Image{
		{Project: "shop", Ref: "docker.io/library/nginx:1.27", SizeBytes: 100, InUse: true},
		{Project: "legacy", Ref: "docker.io/library/redis:7", SizeBytes: 50},
	}}
	h := newHarness(t, withCollector(fake))

	status, body := h.raw(t, http.MethodGet, api.PathImages)
	if status != http.StatusOK {
		t.Fatalf("images = %d: %s", status, body)
	}
	var resp api.ImagesResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Images) != 2 || !resp.Images[0].InUse || resp.Images[1].InUse {
		t.Errorf("images = %+v; want the collector's two rows with their verdicts", resp.Images)
	}
}

// The sweep deletes node state: admin-only, like every other mutation of the
// node itself.
func TestImagesGCIsAdminOnly(t *testing.T) {
	fake := &fakeCollector{}
	h := newAuthHarness(t, withCollector(fake))

	req := h.request(t, http.MethodPost, api.PathImagesGC, nil)
	req.Header.Set("Authorization", "Bearer "+h.token(t, auth.RoleViewer))
	resp, body := h.do(t, req)
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("as viewer = %d, want 403: %s", resp.StatusCode, body)
	}
	if fake.sweeps != 0 {
		t.Errorf("a viewer's request swept (%d sweeps)", fake.sweeps)
	}
}

// A refused collector answers 409 naming the reason rather than sweeping
// past the node's own configuration; the list still works, because a node
// that may not delete may still look.
func TestImagesGCAnswers409OnARefusedCollector(t *testing.T) {
	fake := &fakeCollector{refusal: "the node's default pull policy is \"never\""}
	h := newHarness(t, withCollector(fake))

	status, body := h.raw(t, http.MethodPost, api.PathImagesGC)
	if status != http.StatusConflict {
		t.Fatalf("gc on a refused collector = %d, want 409: %s", status, body)
	}
	if fake.sweeps != 0 {
		t.Errorf("a refused collector swept anyway (%d sweeps)", fake.sweeps)
	}
	if status, _ := h.raw(t, http.MethodGet, api.PathImages); status != http.StatusOK {
		t.Errorf("images list on a refused collector = %d, want 200", status)
	}
}

func TestImagesGCReturnsTheSummary(t *testing.T) {
	fake := &fakeCollector{summary: imagegc.Summary{Removed: 3, ReclaimedBytes: 4096}}
	h := newHarness(t, withCollector(fake))

	status, body := h.raw(t, http.MethodPost, api.PathImagesGC)
	if status != http.StatusOK {
		t.Fatalf("gc = %d: %s", status, body)
	}
	var sum imagegc.Summary
	if err := json.Unmarshal([]byte(body), &sum); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if sum.Removed != 3 || sum.ReclaimedBytes != 4096 {
		t.Errorf("summary = %+v; want the collector's", sum)
	}
}
