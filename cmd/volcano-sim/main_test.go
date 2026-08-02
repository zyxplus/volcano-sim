package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsAllocationPlanAsJSON(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 4}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: train\n    minAvailable: 2\n    replicas: 2\n    request: {gpu: 2}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	if err := run([]string{"-nodes", nodesPath, "-jobs", jobsPath}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"jobName":"train"`) {
		t.Fatalf("output lacks allocation: %s", output.String())
	}
}

func TestRunAcceptsQueuesFile(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	queuesPath := filepath.Join(dir, "queues.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: job\n    queue: high\n    minAvailable: 1\n    replicas: 1\n    request: {gpu: 1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(queuesPath, []byte("queues:\n  - name: high\n    weight: 2\n    capability: {gpu: 1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"-nodes", nodesPath, "-jobs", jobsPath, "-queues", queuesPath}, &output); err != nil {
		t.Fatal(err)
	}
}

func TestExecutableAcceptsNodeAndJobFlags(t *testing.T) {
	dir := t.TempDir()
	nodesPath := filepath.Join(dir, "nodes.yaml")
	jobsPath := filepath.Join(dir, "jobs.yaml")
	if err := os.WriteFile(nodesPath, []byte("nodes:\n  - name: n1\n    capacity: {gpu: 4}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jobsPath, []byte("jobs:\n  - name: train\n    minAvailable: 2\n    replicas: 2\n    request: {gpu: 2}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	command := exec.Command("go", "run", ".", "-nodes", nodesPath, "-jobs", jobsPath)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), `"jobName":"train"`) {
		t.Fatalf("output lacks allocation: %s", output)
	}
}
