# Queue Admission Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Schedule Jobs through weighted Queue ordering while enforcing Queue capability.

**Architecture:** API models Queue ownership and allocation. Loader turns queues YAML into validated objects and supplies default. Scheduler groups Jobs by Queue, orders Queues by weight, and uses the existing Gang journal to atomically update Queue allocation.

**Tech Stack:** Go 1.26 and YAML v3.

## Global Constraints

- Missing Job queue means default.
- Queue capability is a hard resource-vector limit.
- Failed Gang trials do not change Queue.Allocated.

### Task 1: Queue model and loader

**Files:** `pkg/api/types.go`, `pkg/loader/loader.go`, `pkg/loader/loader_test.go`.

- [ ] Write a failing YAML test asserting Queue `{name: inference, weight: 3, capability: {gpu: 4}}` and Job `queue: inference` load into `api.Queue` and `Job.Queue`.
- [ ] Run `go test ./pkg/loader -run TestLoadQueues`; expect missing types/API.
- [ ] Add `Queue{Name, Weight, Capability, Allocated}` and `Job.Queue`; validate unique non-empty Queue names, positive weights, and positive capability; create default Queue when needed.
- [ ] Run `go test ./pkg/loader`; expect PASS.

### Task 2: Weighted Queue scheduling and atomic capability

**Files:** `pkg/scheduler/scheduler.go`, `pkg/scheduler/scheduler_test.go`.

- [ ] Write a failing test with one GPU, queues `high(weight 2)` and `low(weight 1)`, and a 1-GPU Gang in each; assert high allocates first.
- [ ] Write a failing test with Queue capability 4 GPU and an 8-task, minAvailable-8 Gang; assert zero allocation and `queue capability exceeded`.
- [ ] Run `go test ./pkg/scheduler`; expect failure.
- [ ] Change `Run` to accept queues, sort by descending weight/name, group Jobs by Queue, and track journal resource. Before each trial allocation, require `queue.Allocated + journalResource + request` to fit capability. Add journal resource to Queue.Allocated only on Gang commit.
- [ ] Run `go test ./...`; expect PASS.

### Task 3: Command and fixtures

**Files:** `cmd/volcano-sim/main.go`, `testdata/queue/queues.yaml`, `testdata/queue/jobs.yaml`, `testdata/queue/nodes.yaml`.

- [ ] Add required `-queues` flag and a fixture demonstrating high-weight Queue precedence.
- [ ] Run `go test ./... && go run ./cmd/volcano-sim -nodes testdata/queue/nodes.yaml -queues testdata/queue/queues.yaml -jobs testdata/queue/jobs.yaml`; expect a valid JSON plan.
- [ ] Commit with `git commit -m "feat: add weighted queue admission"`.
