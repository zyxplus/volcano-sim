# RunningTask 引用校验设计

## 目标

保证 SessionController 的 RunningTask 快照只引用已存在的 Job、Queue 和 Node。

## 规则

- UpdateRunningTasks 逐项检查 JobName、QueueName、NodeName。
- 任一引用不存在时返回错误，旧快照保持不变。
- 全部合法时一次性替换快照。

## 非目标

- 不自动创建缺失对象。
- 不修复历史脏数据。
- 不改变 Reclaim 规则。

## 验收

- 非法 RunningTask 更新失败且不污染旧状态。
- 合法 RunningTask 更新成功。
- 全量测试通过。
