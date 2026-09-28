package imagegc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/m18h/kanea/internal/reconciler"
	"github.com/m18h/kanea/internal/runtime"
	"github.com/m18h/kanea/internal/store"
)

var testNow = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

const (
	digestA = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	digestC = "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

// old and young are relative to the default 48h MinAge.
func old() time.Time   { return testNow.Add(-72 * time.Hour) }
func young() time.Time { return testNow.Add(-time.Hour) }

// fakeImages is the driver slice: namespaces and their images, in memory.
type fakeImages struct {
	projects []string
	imgs     map[string][]runtime.OwnedImage
	removed  []string
	fail     map[string]error // "project/ref" -> removal error
}

func (f *fakeImages) ImageProjects(context.Context) ([]string, error) {
	return f.projects, nil
}

func (f *fakeImages) ListImages(_ context.Context, project string) ([]runtime.OwnedImage, error) {
	return f.imgs[project], nil
}

func (f *fakeImages) RemoveImage(_ context.Context, project, ref string) error {
	key := project + "/" + ref
	if err := f.fail[key]; err != nil {
		return err
	}
	f.removed = append(f.removed, key)
	return nil
}

func (f *fakeImages) removedSet() map[string]bool {
	out := map[string]bool{}
	for _, r := range f.removed {
		out[r] = true
	}
	return out
}

type harness struct {
	t      *testing.T
	store  store.Store
	images *fakeImages
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	st, err := store.Open(store.Options{Path: t.TempDir() + "/state.db"})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &harness{t: t, store: st, images: &fakeImages{imgs: map[string][]runtime.OwnedImage{}}}
}

func (h *harness) seedService(key string, d reconciler.Desired) {
	h.t.Helper()
	if _, err := store.PutValue(context.Background(), h.store, store.KindService, key, d); err != nil {
		h.t.Fatalf("seed service %s: %v", key, err)
	}
}

func (h *harness) seedAlloc(key string, a reconciler.AllocRecord) {
	h.t.Helper()
	if _, err := store.PutValue(context.Background(), h.store, store.KindAlloc, key, a); err != nil {
		h.t.Fatalf("seed alloc %s: %v", key, err)
	}
}

func (h *harness) collector(mutate ...func(*Config)) *Collector {
	h.t.Helper()
	cfg := Config{
		Store:  h.store,
		Images: h.images,
		Now:    func() time.Time { return testNow },
	}
	for _, m := range mutate {
		m(&cfg)
	}
	c, err := New(cfg)
	if err != nil {
		h.t.Fatalf("new collector: %v", err)
	}
	return c
}

// The core matrix: every stored reference protects its image - the declared
// tag in its short form, the pinned digest, the rollback target, an init
// step's image and a lagging alloc's - while an old unreferenced image in a
// deleted project's namespace is exactly what goes.
func TestSweepDeletesOnlyTheGarbage(t *testing.T) {
	h := newHarness(t)
	h.seedService("shop/web", reconciler.Desired{
		Project: "shop", Service: "web",
		// Short form on purpose: containerd holds the canonical form, and
		// the comparison must survive the difference.
		Image:         "nginx:1.27",
		PinnedImage:   "docker.io/library/nginx@" + digestA,
		RollbackImage: "docker.io/library/nginx@" + digestB,
		Init: []reconciler.InitContainer{
			{Name: "migrate", Image: "ghcr.io/acme/migrate:3"},
		},
	})
	// An alloc still running the image the roll is replacing.
	h.seedAlloc("shop-web-0", reconciler.AllocRecord{
		Project: "shop", Service: "web", Image: "docker.io/library/nginx:1.26",
	})

	h.images.projects = []string{"shop", "legacy"}
	h.images.imgs["shop"] = []runtime.OwnedImage{
		{Ref: "docker.io/library/nginx:1.27", Digest: digestA, CreatedAt: old()},
		{Ref: "docker.io/library/nginx@" + digestA, Digest: digestA, CreatedAt: old()},
		{Ref: "docker.io/library/nginx@" + digestB, Digest: digestB, CreatedAt: old()},
		{Ref: "docker.io/library/nginx:1.26", Digest: digestC, CreatedAt: old()},
		{Ref: "ghcr.io/acme/migrate:3", Digest: digestB, CreatedAt: old()},
	}
	h.images.imgs["legacy"] = []runtime.OwnedImage{
		// A deleted project's images: no record references the repo, so no
		// keep-N applies and the old one collects.
		{Ref: "docker.io/library/redis:7", Digest: digestC, SizeBytes: 100, CreatedAt: old()},
		{Ref: "docker.io/library/redis:6", Digest: digestB, SizeBytes: 50, CreatedAt: young()},
	}

	sum, err := h.collector().Once(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}

	removed := h.images.removedSet()
	if !removed["legacy/docker.io/library/redis:7"] {
		t.Error("the deleted project's old image survived; it is the garbage this exists for")
	}
	if removed["legacy/docker.io/library/redis:6"] {
		t.Error("an image younger than min_age was deleted")
	}
	if len(removed) != 1 {
		t.Errorf("removed %v; want only the deleted project's old image", h.images.removed)
	}
	if sum.Removed != 1 || sum.ReclaimedBytes != 100 {
		t.Errorf("summary = %+v; want 1 removal reclaiming 100 bytes", sum)
	}
}

// keep-N is rollback material for repositories something still references,
// and only those: an unreferenced repository with keep-N would keep a deleted
// project's images forever.
func TestKeepNewestProtectsInUseRepositoriesOnly(t *testing.T) {
	h := newHarness(t)
	h.seedService("shop/web", reconciler.Desired{
		Project: "shop", Service: "web", Image: "nginx:1.27",
	})

	h.images.projects = []string{"shop"}
	h.images.imgs["shop"] = []runtime.OwnedImage{
		// The in-use image is the newest; keep = 2 protects it and the next.
		{Ref: "docker.io/library/nginx:1.27", Digest: digestA, CreatedAt: old()},
		{Ref: "docker.io/library/nginx:1.26", Digest: digestB, CreatedAt: old().Add(-time.Hour)},
		{Ref: "docker.io/library/nginx:1.25", Digest: digestC, CreatedAt: old().Add(-2 * time.Hour)},
	}

	if _, err := h.collector().Once(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	removed := h.images.removedSet()
	if removed["shop/docker.io/library/nginx:1.26"] {
		t.Error("the second-newest image of an in-use repository was deleted; it is the rollback material keep-N exists for")
	}
	if !removed["shop/docker.io/library/nginx:1.25"] {
		t.Error("the third-newest unreferenced image survived keep = 2")
	}
}

// keep = 0 is a legal value and means no per-repository retention at all,
// which is why the field is a pointer ("no data is never zero").
func TestKeepZeroKeepsNothingExtra(t *testing.T) {
	h := newHarness(t)
	h.seedService("shop/web", reconciler.Desired{
		Project: "shop", Service: "web", Image: "nginx:1.27",
	})
	h.images.projects = []string{"shop"}
	h.images.imgs["shop"] = []runtime.OwnedImage{
		{Ref: "docker.io/library/nginx:1.27", Digest: digestA, CreatedAt: old()},
		{Ref: "docker.io/library/nginx:1.26", Digest: digestB, CreatedAt: old()},
	}

	zero := 0
	c := h.collector(func(cfg *Config) { cfg.Keep = &zero })
	if _, err := c.Once(context.Background()); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	removed := h.images.removedSet()
	if !removed["shop/docker.io/library/nginx:1.26"] {
		t.Error("keep = 0 still protected an unreferenced image")
	}
	if removed["shop/docker.io/library/nginx:1.27"] {
		t.Error("the in-use image was deleted under keep = 0; in-use does not depend on keep")
	}
}

// A node whose default pull policy is "never" holds preloaded images that can
// never come back: the collector refuses entirely rather than trusting its
// own carve-outs.
func TestNeverPolicyRefusesTheWholeCollector(t *testing.T) {
	h := newHarness(t)
	h.images.projects = []string{"shop"}
	h.images.imgs["shop"] = []runtime.OwnedImage{
		{Ref: "docker.io/library/nginx:1.20", Digest: digestA, CreatedAt: old()},
	}

	c := h.collector(func(cfg *Config) { cfg.NodePullPolicy = runtime.PullNever })
	if c.Refusal() == "" {
		t.Fatal("no refusal on a never-policy node")
	}
	if _, err := c.Once(context.Background()); !errors.Is(err, ErrRefused) {
		t.Fatalf("Once = %v; want ErrRefused", err)
	}
	if err := c.Run(context.Background()); !errors.Is(err, ErrRefused) {
		t.Fatalf("Run = %v; want ErrRefused", err)
	}
	if len(h.images.removed) != 0 {
		t.Errorf("a refused collector removed %v", h.images.removed)
	}
	// The read half still works: a node that may not delete may still look.
	if _, err := c.List(context.Background()); err != nil {
		t.Errorf("List on a refused collector: %v", err)
	}
}

// The operator's off is a refusal too: a sweep the config forbids is a sweep,
// however it was asked for.
func TestDisabledRefuses(t *testing.T) {
	h := newHarness(t)
	c := h.collector(func(cfg *Config) { cfg.Disabled = true })
	if c.Refusal() == "" {
		t.Fatal("no refusal on a disabled collector")
	}
	if _, err := c.Once(context.Background()); !errors.Is(err, ErrRefused) {
		t.Fatalf("Once = %v; want ErrRefused", err)
	}
}

// Deleting one of two names for the same content reclaims nothing; the bytes
// count once, when the digest's last name goes.
func TestReclaimedBytesCountOncePerDigest(t *testing.T) {
	h := newHarness(t)
	h.images.projects = []string{"legacy"}
	h.images.imgs["legacy"] = []runtime.OwnedImage{
		{Ref: "docker.io/library/redis:7", Digest: digestA, SizeBytes: 100, CreatedAt: old()},
		{Ref: "docker.io/library/redis@" + digestA, Digest: digestA, SizeBytes: 100, CreatedAt: old()},
	}

	sum, err := h.collector().Once(context.Background())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if sum.Removed != 2 {
		t.Errorf("removed %d references; want both names", sum.Removed)
	}
	if sum.ReclaimedBytes != 100 {
		t.Errorf("reclaimed %d bytes; want the content counted once", sum.ReclaimedBytes)
	}
}

// A removal that fails is logged and skipped, and its bytes stay unclaimed:
// the digest still has a holder on disk.
func TestAFailedRemovalDoesNotCountItsBytes(t *testing.T) {
	h := newHarness(t)
	h.images.projects = []string{"legacy"}
	h.images.imgs["legacy"] = []runtime.OwnedImage{
		{Ref: "docker.io/library/redis:7", Digest: digestA, SizeBytes: 100, CreatedAt: old()},
		{Ref: "docker.io/library/redis@" + digestA, Digest: digestA, SizeBytes: 100, CreatedAt: old()},
	}
	h.images.fail = map[string]error{
		"legacy/docker.io/library/redis@" + digestA: errors.New("containerd said no"),
	}

	sum, err := h.collector().Once(context.Background())
	if err != nil {
		t.Fatalf("a failed removal must not fail the sweep: %v", err)
	}
	if sum.Removed != 1 || sum.ReclaimedBytes != 0 {
		t.Errorf("summary = %+v; want 1 removal and no reclaimed bytes while a name survives", sum)
	}
}

// List is the images API's read half: every image, with the verdict.
func TestListReportsTheInUseVerdict(t *testing.T) {
	h := newHarness(t)
	h.seedService("shop/web", reconciler.Desired{
		Project: "shop", Service: "web", Image: "nginx:1.27",
	})
	h.images.projects = []string{"shop"}
	h.images.imgs["shop"] = []runtime.OwnedImage{
		{Ref: "docker.io/library/nginx:1.27", Digest: digestA, CreatedAt: old()},
		{Ref: "docker.io/library/nginx:1.20", Digest: digestB, CreatedAt: old()},
	}

	images, err := h.collector().List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(images) != 2 {
		t.Fatalf("listed %d images; want 2", len(images))
	}
	byRef := map[string]Image{}
	for _, img := range images {
		byRef[img.Ref] = img
	}
	if !byRef["docker.io/library/nginx:1.27"].InUse {
		t.Error("the declared image is not marked in use")
	}
	if byRef["docker.io/library/nginx:1.20"].InUse {
		t.Error("an unreferenced image is marked in use")
	}
}

// A stored reference that does not parse cannot be protected, but it must
// not fail the sweep either: it can never match a containerd name anyway.
func TestAMalformedStoredReferenceIsSkipped(t *testing.T) {
	h := newHarness(t)
	h.seedService("shop/web", reconciler.Desired{
		Project: "shop", Service: "web", Image: "NOT a ref!!",
	})
	h.images.projects = nil

	if _, err := h.collector().Once(context.Background()); err != nil {
		t.Fatalf("sweep over a malformed reference: %v", err)
	}
}

func TestNegativeKeepIsRefusedAtNew(t *testing.T) {
	h := newHarness(t)
	neg := -1
	_, err := New(Config{Store: h.store, Images: h.images, Keep: &neg})
	if err == nil || !strings.Contains(err.Error(), "negative") {
		t.Fatalf("New with keep -1 = %v; want a refusal naming it", err)
	}
}
