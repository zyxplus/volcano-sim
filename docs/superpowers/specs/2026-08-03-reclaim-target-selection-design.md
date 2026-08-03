# Reclaim Target Selection 设计

## 目标

`RunWithReclaim` 每次只为一个 pending Job 做 reclaim 决策。目标 Job 由现有 `OrderJobs` 选择，保持 DRF 与名称排序语义；`RunSession` 在每轮重新构造 pending 集合，因此一次 reclaim 成功后可以继续处理其他 Job。

## 非目标

- 不在一次 reclaim 中同时满足多个 Job。
- 不改变 victim 排序、队列 overused/reclaimable 判断或优先级门槛。
- 不引入 `RoundPlan` / `SessionPlan` 新类型。
- 不重构队列状态回滚模型。

## 行为

1. `RunWithReclaim` 收到空 Job 列表时直接返回空计划。
2. 非空时先用 `OrderJobs` 选出一个目标 Job。
3. 需求量、等待优先级、拓扑约束和分配尝试只针对该目标 Job。
4. reclaim 失败时不保留 Eviction，节点状态回滚。
5. `RunSession` 每轮从未完成 Job 中重新选择目标，直到普通调度与 reclaim 都无进展。

## 验收

- 第一个 pending Job 无法 reclaim、第二个 pending Job 可以 reclaim 时，第二个 Job 能成功分配。
- 空 Job 列表不会 panic。
- 现有调度、回收、回滚测试全部通过。
