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
