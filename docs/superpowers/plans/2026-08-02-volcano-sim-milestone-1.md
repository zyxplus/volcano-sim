# Volcano-Sim Milestone 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a Go, YAML-driven in-memory scheduler that orders jobs by DRF and atomically schedules gang jobs.

**Architecture:** `api` owns resource and scheduling data. `loader` decodes and validates YAML into those data types. `scheduler` performs deterministic DRF ordering and uses a per-job allocation journal to either commit every tentative allocation or restore every affected node. The command only composes loader and scheduler.

**Tech Stack:** Go 1.22+, standard library, `gopkg.in/yaml.v3`.

## Global Constraints

- Resource quantities use `map[string]float64`; no CPU, memory, or GPU-specific fields.
- Resource arithmetic returns new values and never mutates either operand.
- A gang job commits no allocation unless its tentative count is at least `minAvailable`.
- Equal DRF shares and equal node choices sort by name for deterministic output.
- Invalid input returns an error; ordinary capacity shortage becomes an unschedulable-plan reason.

---

### Task 1: Bootstrap the Go module and resource vector

**Files:**
- Create: `go.mod`
- Create: `pkg/api/resource.go`
- Create: `pkg/api/resource_test.go`

**Consumes:** none.

**Produces:** `type Resource map[string]float64`, `NewResource`, `Add`, `Sub`, `Fits`, `DominantShare`.

- [ ] **Step 1: Write the failing resource test**

```go
func TestResourceOperationsSupportArbitraryDimensions(t *testing.T) {
	allocated := NewResource(map[string]float64{"cpu": 2, "rdma": 1})
	request := NewResource(map[string]float64{"cpu": 1, "gpu": 2})
	got := allocated.Add(request)
	if !got.Fits(NewResource(map[string]float64{"cpu": 3, "gpu": 2, "rdma": 1})) {
		t.Fatal("combined resource should fit exact capacity")
	}
	if allocated["gpu"] != 0 {
		t.Fatal("Add must not mutate its receiver")
	}
}
```

- [ ] **Step 2: Run the test and verify it fails because `NewResource` is undefined**

Run: `go test ./pkg/api -run TestResourceOperationsSupportArbitraryDimensions`

Expected: compile failure mentioning `undefined: NewResource`.

- [ ] **Step 3: Implement the minimal resource API**

```go
package api

type Resource map[string]float64

func NewResource(values map[string]float64) Resource { /* copy values */ }
func (r Resource) Add(other Resource) Resource       { /* copied sum */ }
func (r Resource) Sub(other Resource) Resource       { /* copied difference */ }
func (r Resource) Fits(capacity Resource) bool       { /* each requested key <= capacity */ }
func (r Resource) DominantShare(total Resource) float64 { /* maximum non-zero total share */ }
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./pkg/api`

Expected: `ok .../pkg/api`.

### Task 2: Model cluster state and plan results

**Files:**
- Create: `pkg/api/types.go`
- Create: `pkg/api/types_test.go`

**Consumes:** `Resource` from `pkg/api/resource.go`.

**Produces:** `Node`, `Job`, `Task`, `Allocation`, `AllocationPlan`, and `ClusterTotal(nodes []Node) Resource`.

- [ ] **Step 1: Write the failing cluster-total test**

```go
func TestClusterTotalSumsNodeCapacity(t *testing.T) {
	nodes := []Node{{Name: "n1", Capacity: NewResource(map[string]float64{"gpu": 2})}, {Name: "n2", Capacity: NewResource(map[string]float64{"gpu": 3, "cpu": 4})}}
	got := ClusterTotal(nodes)
	if got["gpu"] != 5 || got["cpu"] != 4 {
		t.Fatalf("unexpected total: %#v", got)
	}
}
```

- [ ] **Step 2: Run the test and verify it fails because `Node` is undefined**

Run: `go test ./pkg/api -run TestClusterTotalSumsNodeCapacity`

Expected: compile failure mentioning `undefined: Node`.

- [ ] **Step 3: Implement the public scheduling data types**

```go
type Node struct { Name string; Capacity Resource; Idle Resource }
type Job struct { Name string; MinAvailable int; Replicas int; Request Resource; Allocated Resource }
type Task struct { JobName string; Index int; Request Resource }
type Allocation struct { JobName string; TaskIndex int; NodeName string }
type AllocationPlan struct { Allocations []Allocation; Unschedulable map[string]string }
func ClusterTotal(nodes []Node) Resource { /* sum Capacity */ }
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./pkg/api`

Expected: `ok .../pkg/api`.

### Task 3: Implement DRF ordering and atomic gang allocation

**Files:**
- Create: `pkg/scheduler/scheduler.go`
- Create: `pkg/scheduler/scheduler_test.go`

**Consumes:** API types and `ClusterTotal`.

**Produces:** `New(nodes []api.Node) *Scheduler`, `OrderJobs(jobs []api.Job) []api.Job`, and `Run(jobs []api.Job) api.AllocationPlan`.

- [ ] **Step 1: Write a failing DRF-order test**

```go
func TestOrderJobsUsesLowestDominantShareFirst(t *testing.T) {
	s := New([]api.Node{{Name: "n", Capacity: api.NewResource(map[string]float64{"cpu": 100, "gpu": 10})}})
	jobs := []api.Job{
		{Name: "gpu-heavy", Allocated: api.NewResource(map[string]float64{"gpu": 5})},
		{Name: "cpu-heavy", Allocated: api.NewResource(map[string]float64{"cpu": 20})},
	}
	got := s.OrderJobs(jobs)
	if got[0].Name != "cpu-heavy" { t.Fatalf("got %q first", got[0].Name) }
}
```

- [ ] **Step 2: Run it and verify it fails because `New` is undefined**

Run: `go test ./pkg/scheduler -run TestOrderJobsUsesLowestDominantShareFirst`

Expected: compile failure mentioning `undefined: New`.

- [ ] **Step 3: Implement only DRF ordering, then rerun its test**

Use `sort.SliceStable`, compare `job.Allocated.DominantShare(clusterTotal)`, and use `Name` as the tie-breaker.

Run: `go test ./pkg/scheduler -run TestOrderJobsUsesLowestDominantShareFirst`

Expected: `PASS`.

- [ ] **Step 4: Write the failing gang rollback test**

```go
func TestRunRollsBackJobWhenGangCannotReachMinimum(t *testing.T) {
	s := New([]api.Node{{Name: "n1", Capacity: api.NewResource(map[string]float64{"gpu": 2}), Idle: api.NewResource(map[string]float64{"gpu": 2})}})
	plan := s.Run([]api.Job{{Name: "four-gpu-tasks", MinAvailable: 4, Replicas: 4, Request: api.NewResource(map[string]float64{"gpu": 1})}})
	if len(plan.Allocations) != 0 { t.Fatalf("partial gang allocation leaked: %#v", plan.Allocations) }
	if plan.Unschedulable["four-gpu-tasks"] != "insufficient resources for minAvailable" { t.Fatal("missing reason") }
}
```

- [ ] **Step 5: Run it and verify it fails because `Run` is undefined**

Run: `go test ./pkg/scheduler -run TestRunRollsBackJobWhenGangCannotReachMinimum`

Expected: compile failure mentioning `Run` or a failing assertion after a stub.

- [ ] **Step 6: Implement the allocation journal and gang commit rule**

For every trial allocation, decrement the selected node's `Idle` by the task request and append an `api.Allocation` to a local journal. Choose the first name-sorted node where `Request.Fits(Idle)`. If journal length is below `MinAvailable`, add each request back to its recorded node and omit all journal entries from the plan. Otherwise append the journal to the plan.

- [ ] **Step 7: Run all scheduler tests**

Run: `go test ./pkg/scheduler`

Expected: `ok .../pkg/scheduler`.

### Task 4: Add YAML input, fixtures, and the command-line program

**Files:**
- Create: `pkg/loader/loader.go`
- Create: `pkg/loader/loader_test.go`
- Create: `cmd/volcano-sim/main.go`
- Create: `testdata/basic/nodes.yaml`
- Create: `testdata/basic/jobs.yaml`

**Consumes:** API types and scheduler `New`/`Run`.

**Produces:** `Load(nodesPath, jobsPath string) ([]api.Node, []api.Job, error)` and command `go run ./cmd/volcano-sim -nodes ... -jobs ...`.

- [ ] **Step 1: Write a failing loader test using temporary YAML files**

```go
func TestLoadBuildsNodesAndJobs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "nodes.yaml"), []byte("nodes:\n  - name: n1\n    capacity: {gpu: 4}\n"), 0o600); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "jobs.yaml"), []byte("jobs:\n  - name: train\n    minAvailable: 2\n    replicas: 2\n    request: {gpu: 2}\n"), 0o600); err != nil { t.Fatal(err) }
	nodes, jobs, err := Load(filepath.Join(dir, "nodes.yaml"), filepath.Join(dir, "jobs.yaml"))
	if err != nil || len(nodes) != 1 || len(jobs) != 1 { t.Fatalf("nodes=%d jobs=%d err=%v", len(nodes), len(jobs), err) }
}
```

- [ ] **Step 2: Run the test and verify it fails because `Load` is undefined**

Run: `go test ./pkg/loader -run TestLoadBuildsNodesAndJobs`

Expected: compile failure mentioning `undefined: Load`.

- [ ] **Step 3: Add `gopkg.in/yaml.v3` and implement validated loading**

Decode wrapper structs with `nodes` and `jobs`; reject empty names, non-positive resource quantities, `minAvailable <= 0`, and `minAvailable > replicas`. Initialize every node's `Idle` as a copy of `Capacity`.

- [ ] **Step 4: Implement CLI JSON output and fixture files**

Parse `-nodes` and `-jobs` flags, call `loader.Load`, call `scheduler.New(nodes).Run(jobs)`, and encode the plan to stdout with `json.NewEncoder`.

- [ ] **Step 5: Run full verification**

Run: `go test ./... && go run ./cmd/volcano-sim -nodes testdata/basic/nodes.yaml -jobs testdata/basic/jobs.yaml`

Expected: all packages pass, then a JSON plan with two allocations for `train`.
