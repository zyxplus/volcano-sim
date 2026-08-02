package loader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadBuildsNodesAndJobs(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 4}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: train\n    minAvailable: 2\n    replicas: 2\n    request: {gpu: 2}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nodes, jobs, err := Load(nodesPath, jobsPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 || len(jobs) != 1 {
		t.Fatalf("nodes=%d jobs=%d, want one each", len(nodes), len(jobs))
	}
	if nodes[0].Idle["gpu"] != 4 {
		t.Fatalf("idle GPU = %v, want 4", nodes[0].Idle["gpu"])
	}
}

func TestLoadReadsRequiredFabricTopology(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 4}\n    labels: {fabric-id: fabric-a, gpu-model: c550}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: train\n    minAvailable: 2\n    replicas: 2\n    request: {gpu: 2}\n    topology: {sameFabric: Required, gpuModel: c550}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	nodes, jobs, err := Load(nodesPath, jobsPath)
	if err != nil {
		t.Fatal(err)
	}
	if nodes[0].Labels["fabric-id"] != "fabric-a" {
		t.Fatal("fabric label missing")
	}
	if jobs[0].Topology == nil || jobs[0].Topology.SameFabric != "Required" {
		t.Fatal("topology missing")
	}
}

func TestLoadRejectsRequiredTopologyWithoutGPUModel(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 4}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: train\n    minAvailable: 1\n    replicas: 1\n    request: {gpu: 1}\n    topology: {sameFabric: Required}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, _, err := Load(nodesPath, jobsPath); err == nil {
		t.Fatal("Load accepted Required topology without gpuModel")
	}
}

func TestLoadQueues(t *testing.T) {
	dir := t.TempDir()
	queuesPath := filepath.Join(dir, "queues.yaml")
	if err := os.WriteFile(queuesPath, []byte("queues:\n  - name: inference\n    weight: 3\n    capability: {gpu: 4}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	queues, err := LoadQueues(queuesPath)
	if err != nil {
		t.Fatal(err)
	}
	if queues[0].Name != "inference" || queues[0].Weight != 3 {
		t.Fatalf("unexpected queue: %#v", queues[0])
	}
}

func TestLoadQueuesReadsGuaranteeAndReclaimable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "queues.yaml")
	if err := os.WriteFile(path, []byte("queues:\n  - name: training\n    weight: 1\n    capability: {gpu: 8}\n    guarantee: {gpu: 4}\n    reclaimable: true\n"), 0o600); err != nil { t.Fatal(err) }
	queues, err := LoadQueues(path)
	if err != nil { t.Fatal(err) }
	if queues[0].Guarantee["gpu"] != 4 || !queues[0].Reclaimable { t.Fatalf("unexpected queue: %#v", queues[0]) }
}

func TestLoadAssignsDefaultQueueToJobWithoutQueue(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: job\n    minAvailable: 1\n    replicas: 1\n    request: {gpu: 1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, jobs, err := Load(nodesPath, jobsPath)
	if err != nil {
		t.Fatal(err)
	}
	if jobs[0].Queue != "default" {
		t.Fatalf("queue = %q, want default", jobs[0].Queue)
	}
}
