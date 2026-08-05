# Duplicate Queue 校验设计

## 目标

防止同名 Queue 被调度两次，避免同一 Job 获得重复 Allocation。

## 规则

- Queue 名称必须唯一。
- 重复名称对应的 Job 标记为 `queue duplicated`，不参与调度。
- 其他唯一 Queue 继续正常调度。

## 非目标

- 不自动合并重复 Queue。
- 不改变 Queue 权重或资源限制。

## 验收

- 重复 Queue 下 Job 没有 Allocation，并有明确原因。
- 唯一 Queue 下 Job 正常调度。
- 不会产生重复 Allocation。
