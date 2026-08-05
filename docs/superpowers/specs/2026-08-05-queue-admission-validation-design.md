# Queue Admission 校验设计

## 目标

在调度入口识别 Job 引用的未知 Queue，并返回明确的 `queue not found` 原因，避免 Job 被静默忽略。

## 规则

- 已知 Queue 的 Job 按现有逻辑调度。
- 未知 Queue 的 Job 不产生 Allocation。
- 未知 Queue 的 Job 在 `Unschedulable[job.Name]` 中记录 `queue not found`。
- 空 Queue 名称也视为未知 Queue，除非调用方显式提供同名 Queue。

## 非目标

- 不自动创建 Queue。
- 不改变 Queue 权重、Capability、Guarantee 或 Reclaimable 语义。
- 不改变 Job 的原始 Queue 字段。

## 验收

- 合法 Job 正常调度。
- 未知 Queue Job 有明确失败原因。
- 合法与非法 Job 混合时互不影响。
