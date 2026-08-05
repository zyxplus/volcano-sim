# RunningTask 唯一性设计

## 目标

保证运行态快照中每个 Job 的 TaskIndex 唯一，避免资源被重复计数。

## 规则

- `(JobName, TaskIndex)` 作为 RunningTask 的唯一身份。
- UpdateRunningTasks 发现重复身份时拒绝整批更新。
- 旧 RunningTask 快照保持不变。

## 非目标

- 不自动合并重复 Task。
- 不改变 TaskIndex 的生成规则。

## 验收

- 重复 Task 更新失败且状态不变。
- 不同 TaskIndex 正常更新。
- 全量测试通过。
