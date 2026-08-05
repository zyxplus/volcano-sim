# Loader Duplicate Job 校验设计

## 目标

在加载 Job 配置时拒绝重复名称，保证 Allocation 与 Unschedulable 能唯一关联到一个 Job。

## 规则

- Job 名称必须非空且唯一。
- 重复名称返回包含名称的错误。
- 调度器对绕过 Loader 的输入保持现有行为。

## 非目标

- 不自动重命名 Job。
- 不改变 Job 的其他字段校验。

## 验收

- 重复 Job YAML 加载失败。
- 唯一 Job YAML 正常加载。
- 全量测试通过。
