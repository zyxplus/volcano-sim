# Reclaim 队列状态回滚设计

## 目标

Reclaim 是试算过程。若驱逐后等待 Gang 仍无法达到 `minAvailable`，必须恢复节点 `Idle` 与所有受影响源队列的 `Allocated`；只有后续分配成功时才保留队列扣减。

## 非目标

- 不改变 victim 选择顺序或优先级门槛。
- 不改变 overused/reclaimable 判断。
- 不新增事务类型。
- 不修改 Job 已提交的分配状态。

## 行为

记录每次试算对源队列的资源扣减。分配失败时按记录逐项恢复；分配成功时保留扣减，并返回对应 Eviction。

## 验收

- 单个 victim 导致 Gang 失败时，源队列 `Allocated` 恢复原值。
- 多个 victim 导致 Gang 失败时，所有源队列状态都恢复。
- reclaim 成功时，源队列扣减保持。
- 全量测试通过。
