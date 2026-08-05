# RemoveJob RunningTask 保护设计

## 目标

避免删除仍有运行任务的 Job，防止 SessionController 产生悬空 RunningTask。

## 规则

- RemoveJob 前检查当前 RunningTask 的 JobName。
- 仍有任务时返回错误，不修改 Job 或任务快照。
- 任务通过 UpdateRunningTasks 清除后，允许删除 Job。

## 非目标

- 不自动驱逐 Job 的任务。
- 不改变 Scheduler 的 Reclaim 逻辑。

## 验收

- 有任务 Job 删除失败且状态不变。
- 空闲 Job 可以删除。
- 清除任务后 Job 可以删除。
