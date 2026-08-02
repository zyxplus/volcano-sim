# Dynamic DRF Implementation Plan

**Goal:** Reorder remaining jobs in a Queue after each successful Gang commit.

1. Write a scheduler test with two equal-share jobs; assert the second job is selected after the first commit.
2. Change Queue scheduling from a `for range OrderJobs(...)` loop to a pending-job loop that calls `OrderJobs` before each selection.
3. On a committed journal, add its resource vector to the selected Job.Allocated; do not update it after rollback.
4. Run `go test ./...`, commit, and merge after verification.
