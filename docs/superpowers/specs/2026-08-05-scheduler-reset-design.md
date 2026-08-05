# Scheduler.Reset 设计

## 目标

允许同一个 Scheduler 在多个独立 Session 之间重置节点运行时状态。

## 规则

- `Scheduler` 保存创建时的 NodeSpec 副本。
- `Reset()` 根据 NodeSpec 重建 NodeState，恢复 `Idle = Capacity`。
- Reset 不修改外部 NodeSpec、Job 或 Queue 对象。
- `total` 保持不变。

## 非目标

- 不实现 Job/Node 动态事件循环。
- 不自动重置 Job.ScheduledReplicas 或 Queue.Allocated。
- 不改变调度算法。

## 验收

- 一次调度消耗资源后，Reset 再次调度可获得完整节点资源。
- Reset 后节点状态与创建时一致。
- 全量测试通过。
