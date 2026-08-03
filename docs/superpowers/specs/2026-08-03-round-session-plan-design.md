# RoundPlan / SessionPlan 分层设计

## 目标

区分单轮调度决策与整个 Session 的聚合结果，同时保持现有 `RunSession` 返回 `api.AllocationPlan` 的兼容性。

## 设计

- `api.RoundPlan` 表示一次普通调度或 reclaim 尝试，包含 `Kind`、Allocation、Eviction 和 Unschedulable。
- `api.SessionPlan` 保存有序的 `[]RoundPlan`。
- `SessionPlan.Summary()` 将所有轮次聚合为旧的 `AllocationPlan`。
- 新增 `Scheduler.RunSessionDetailed` 返回 `SessionPlan`；现有 `RunSession` 调用它并返回 Summary。
- 零进展轮次也保留，便于解释 Session 为什么结束。

## 非目标

- 不改变调度排序、Gang、Fabric、Queue 或 reclaim 决策。
- 不引入 Kubernetes Bind/Evict 执行器。
- 不修改 CLI 默认输出格式。

## 验收

- 普通调度与 reclaim 结果分别出现在对应 RoundPlan。
- SessionPlan 的 Summary 与原 RunSession 结果一致。
- 无进展结束轮次仍保留失败原因。
- 全量测试通过。
