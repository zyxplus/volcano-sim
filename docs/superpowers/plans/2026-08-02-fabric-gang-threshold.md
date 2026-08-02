# Fabric Gang Threshold Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Required Fabric feasibility use the Gang `minAvailable` threshold.

**Architecture:** Only `selectCandidateNodes` changes: it computes the resource vector needed to launch the gang, then applies its existing Fabric grouping and candidate filtering. The journal remains unchanged.

**Tech Stack:** Go 1.26 and the existing Go test suite.

## Global Constraints

- Required Fabric capacity is `MinAvailable × Request`.
- A successful plan may contain fewer allocations than Replicas but never fewer than MinAvailable.
- Existing no-topology and insufficient-Fabric behavior remains unchanged.

---

### Task 1: Align Fabric selection with Gang readiness

**Files:**
- Modify: `pkg/scheduler/scheduler_test.go`
- Modify: `pkg/scheduler/scheduler.go`

**Consumes:** `api.Job.MinAvailable`, `api.Job.Replicas`, and `api.Resource.Add`.

**Produces:** a Fabric requirement based on MinAvailable.

- [ ] **Step 1: Write the failing test**

```go
func TestRunAllowsRequiredGangWhenFabricFitsMinAvailable(t *testing.T) {
	nodes := []api.Node{{Name: "a", Capacity: api.NewResource(map[string]float64{"gpu": 4}), Labels: map[string]string{"fabric-id": "fabric-a", "gpu-model": "c550"}}}
	job := api.Job{Name: "elastic", Replicas: 8, MinAvailable: 4, Request: api.NewResource(map[string]float64{"gpu": 1}), Topology: &api.Topology{GPUModel: "c550", SameFabric: "Required"}}
	plan := New(nodes).Run([]api.Job{job})
	if len(plan.Allocations) != 4 { t.Fatalf("allocation count = %d, want 4", len(plan.Allocations)) }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./pkg/scheduler -run TestRunAllowsRequiredGangWhenFabricFitsMinAvailable`

Expected: failure with `no single fabric has enough capacity`, because the implementation currently requires 8 GPU.

- [ ] **Step 3: Implement the minimal change**

```go
need := api.NewResource(nil)
for replica := 0; replica < job.MinAvailable; replica++ {
	need = need.Add(job.Request)
}
```

Replace the loop in `selectCandidateNodes` that currently uses `job.Replicas`.

- [ ] **Step 4: Verify GREEN**

Run: `go test ./...`

Expected: every package passes.

- [ ] **Step 5: Commit**

```bash
git add pkg/scheduler/scheduler.go pkg/scheduler/scheduler_test.go
git commit -m "fix: align fabric capacity with gang threshold"
```
