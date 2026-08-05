# SessionController 运行态重建设计

## 目标

把 `RunningTask` 作为跨 Session 的运行态事实来源，在每次 Run 前重建 Job、Queue 和 Node 的已用资源。

## 规则

- 按 JobName 汇总 Task 数量和 Request，更新 `ScheduledReplicas` / `Allocated`。
- 按 QueueName 汇总 Request，更新 Queue.Allocated。
- 按 NodeName 汇总 Request，计算 Node Idle。
- 已有运行 Task 的 Job 不会在下一次 Session 被重复分配。

## 非目标

- 不记录历史版本或事件日志。
- 不处理未知 Job/Queue/Node 的持久化修复，只保留当前调度器的兼容行为。
- 不改变单次 Scheduler 的算法。

## 验收

- Job A 已有运行 Task 时，下一次 Run 只为未完成 Job 分配。
- Queue 和 Node 资源与 RunningTask 汇总一致。
- 全量测试通过。
