# Scheduling Reasons 设计

AllocationPlan.Unschedulable 是统一的 `jobName -> reason` 可观测输出。Reclaim 发现 Queue 条件满足但所有候选 victim 都因 priority 门槛被拒绝时，记录 `reclaim blocked by priority`；这与普通 `insufficient resources for minAvailable` 区分。

CLI 保持现有 JSON 编码，无新增协议。
