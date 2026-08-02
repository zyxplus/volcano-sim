# Reclaim Dry-Run 设计

当等待 Gang 无法仅用空闲资源达到 minAvailable 时，调度器从 `overused && reclaimable` Queue 中按 Task 名称选择 victim。Evict 与 Allocate 都记录在同一 journal；只有新 Gang 达标时才输出两类决策，否则同时回滚 Node 和 Queue 资源。

第一阶段不调用 Kubernetes API，不处理 priority/PDB。AllocationPlan 新增 Evictions。
