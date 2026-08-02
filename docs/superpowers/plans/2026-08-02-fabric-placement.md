# Fabric Placement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prevent a Required-topology gang job from being allocated across multiple GPU fabrics.

**Architecture:** API types carry opaque node labels and an optional job topology. The scheduler selects a complete feasible Fabric before it creates a Gang allocation journal; the existing journal remains the sole mechanism that changes and rolls back `Node.Idle`.

**Tech Stack:** Go 1.26, standard library, `gopkg.in/yaml.v3`.

## Global Constraints

- Only `sameFabric: Required` changes placement; a missing topology keeps existing behavior.
- A matching Fabric must fit `Replicas × Request` using current Node.Idle before any trial allocation.
- A failed Fabric selection must not alter Node.Idle or emit allocations.
- Fabric and node selection are deterministic: fabric-id, then node name.

---

### Task 1: Model and validate topology YAML

**Files:**
- Modify: `pkg/api/types.go`
- Modify: `pkg/loader/loader.go`
- Modify: `pkg/loader/loader_test.go`

**Consumes:** `api.Node`, `api.Job`, and YAML loading.

**Produces:** `Node.Labels map[string]string`; `Topology{GPUModel, SameFabric}`; `Job.Topology *Topology`.

- [ ] **Step 1: Write the failing loader test**

```go
func TestLoadReadsRequiredFabricTopology(t *testing.T) {
	// YAML node: labels {fabric-id: fabric-a, gpu-model: c550}
	// YAML job: topology {sameFabric: Required, gpuModel: c550}
	nodes, jobs, err := Load(nodesPath, jobsPath)
	if err != nil { t.Fatal(err) }
	if nodes[0].Labels["fabric-id"] != "fabric-a" { t.Fatal("fabric label missing") }
	if jobs[0].Topology == nil || jobs[0].Topology.SameFabric != "Required" { t.Fatal("topology missing") }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./pkg/loader -run TestLoadReadsRequiredFabricTopology`

Expected: compile failure because `Node.Labels` and `Job.Topology` do not exist.

- [ ] **Step 3: Implement model and validation**

```go
type Topology struct {
	GPUModel string `yaml:"gpuModel" json:"gpuModel"`
	SameFabric string `yaml:"sameFabric" json:"sameFabric"`
}

// Add Labels map[string]string to Node and Topology *Topology to Job.
// In loader, reject Required topology with empty GPUModel, and reject a
// non-empty SameFabric other than Required.
```

- [ ] **Step 4: Verify GREEN**

Run: `go test ./pkg/loader`

Expected: `ok github.com/zhouyingxiao/volcano-sim/pkg/loader`.

### Task 2: Preselect a feasible Fabric before Gang allocation

**Files:**
- Modify: `pkg/scheduler/scheduler.go`
- Modify: `pkg/scheduler/scheduler_test.go`

**Consumes:** `api.Job.Topology`, `api.Node.Labels`, and Resource vector arithmetic.

**Produces:** `selectCandidateNodes(job api.Job) ([]int, string)` used by `Run`.

- [ ] **Step 1: Write the failing topology test**

```go
func TestRunRejectsRequiredGangSplitAcrossFabrics(t *testing.T) {
	nodes := []api.Node{
		{Name: "a", Labels: map[string]string{"fabric-id": "a", "gpu-model": "c550"}, Capacity: api.NewResource(map[string]float64{"gpu": 16})},
		{Name: "b", Labels: map[string]string{"fabric-id": "b", "gpu-model": "c550"}, Capacity: api.NewResource(map[string]float64{"gpu": 16})},
	}
	job := api.Job{Name: "large", MinAvailable: 24, Replicas: 24, Request: api.NewResource(map[string]float64{"gpu": 1}), Topology: &api.Topology{GPUModel: "c550", SameFabric: "Required"}}
	plan := New(nodes).Run([]api.Job{job})
	if len(plan.Allocations) != 0 || plan.Unschedulable["large"] != "no single fabric has enough capacity" { t.Fatalf("unexpected plan: %#v", plan) }
}
```

- [ ] **Step 2: Verify RED**

Run: `go test ./pkg/scheduler -run TestRunRejectsRequiredGangSplitAcrossFabrics`

Expected: the test fails because current `Run` allocates across both fabrics.

- [ ] **Step 3: Implement Fabric selection and candidate-aware node search**

```go
// selectCandidateNodes returns all node indexes when topology is absent.
// For Required topology, group matching gpu-model nodes by fabric-id, sum Idle
// per group, choose the lexicographically first group fitting Replicas*Request,
// and return only its indexes. Return reason when none fits.
func (s *Scheduler) findNode(request api.Resource, candidates []int) int
```

- [ ] **Step 4: Verify full behavior**

Run: `go test ./... && go run ./cmd/volcano-sim -nodes testdata/basic/nodes.yaml -jobs testdata/basic/jobs.yaml`

Expected: all tests pass and the existing topology-free fixture still outputs two `train` allocations.
