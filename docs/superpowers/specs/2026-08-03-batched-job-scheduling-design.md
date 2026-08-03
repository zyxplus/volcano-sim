# Batched Job Scheduling 设计

## 目标

让 Job 在多轮调度中逐批获得 replica，使提交后的 dominant share 能影响下一次 DRF 选择。

## 语义

Job 增加 `batchSize` 与 `scheduledReplicas`。未启动 Job 的首次提交必须达到 `minAvailable`，且最多提交 `max(minAvailable, batchSize)` 个 replica。已启动 Job 每轮最多提交 `batchSize` 个剩余 replica。完成全部 replicas 前 Job 留在候选集。

每个 batch 成功后更新 Job.Allocated 与 ScheduledReplicas，再重新计算同 Queue 剩余 Job 的 DRF 顺序。

未完成 Job 留在候选集。若完整遍历候选集后没有任一 Job 成功提交 batch，Session 结束并保留不可调度原因，避免无限空转。

## 验收

两个未完成 Job 都可继续调度；A 提交一个 batch 后 dominant share 上升，下一次 batch 必须优先选择 B。
