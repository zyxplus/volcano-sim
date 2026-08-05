# Loader Duplicate Queue 校验设计

## 目标

在加载 Queue 配置时提前拒绝重复名称，尽早发现静态配置错误。

## 规则

- `LoadQueues` 维护已见名称集合。
- 重复名称返回包含 Queue 名称的错误。
- 调度器保留运行时重复校验，覆盖绕过 Loader 的调用方。

## 非目标

- 不自动合并同名 Queue。
- 不改变 Queue 的其他字段校验。

## 验收

- 重复 Queue YAML 加载失败。
- 唯一 Queue YAML 加载成功。
- 全量测试通过。
