# ValidationIssue 错误分级设计

## 目标

为调度结果增加统一、可选的校验问题模型，区分配置 Fatal 与单 Job 的 JobError，同时保持旧字段兼容。

## 设计

- `api.ValidationIssue` 包含 Severity、Object、Name、Message。
- Loader 现有 `error` 继续表示 Fatal，不改变其 API。
- Scheduler 将未知/重复 Queue 写入 `AllocationPlan.Issues`，Severity 为 JobError；同时保留 `Unschedulable`。
- `RoundPlan` 和 `SessionPlan.Summary()` 传递并聚合 Issues。
- `issues` 使用 `omitempty`，没有问题时不改变 CLI JSON 结构。

## 非目标

- 不把所有资源不足都改成 ValidationIssue。
- 不引入 Warning 生成逻辑。
- 不改变调度结果。

## 验收

- 未知 Queue Job 具有 JobError issue。
- RunSession 聚合 RoundPlan issues。
- Loader Fatal error 行为保持不变。
