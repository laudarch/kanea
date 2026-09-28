// Package runtime is the containerd driver: image pull (TLS-only registries,
// digest pinning), task lifecycle, per-alloc netns (CNI call), cgroup metrics
// (single /v1/metrics scrape), stdout/stderr capture with non-blocking drains,
// and the image list/remove surface internal/imagegc sweeps through (§5.2.4,
// v1.111; disk watermark alerts are still owed). Workload hardening defaults
// are applied here (drop caps, no-new-privileges, seccomp: PRD §5.2.4, §14 A05).
package runtime
