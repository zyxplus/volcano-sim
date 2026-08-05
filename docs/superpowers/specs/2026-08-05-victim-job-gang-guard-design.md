# Victim Job Gang 保护设计

## 目标

Reclaim 选择 victim 时，保护 victim 所属 Job 的 `MinAvailable`，避免部分驱逐破坏其 Gang 约束。

## 数据模型

`RunningTask` 增加可选的 `JobMinAvailable` 与 `JobRunningReplicas`。当二者均为 0 时视为旧调用方未提供状态，保持兼容。

## 规则

对同一 victim Job 记录本轮已选择的驱逐数。只有满足：

`JobRunningReplicas - selectedVictims - 1 >= JobMinAvailable`

时才允许再选择该 victim。

## 非目标

- 不改变 waiting Job 的 Gang 逻辑。
- 不自动整 Job 驱逐。
- 不改变 victim 优先级、Queue 资格和回滚语义。

## 验收

- 驱逐后 victim Job 仍满足 MinAvailable。
- 同一 victim Job 的多个 Task 按累计驱逐数检查。
- 未提供新状态字段的旧 RunningTask 保持原行为。
