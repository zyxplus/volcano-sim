# NodeSpec / NodeState 分离设计

## 目标

把 Node 的静态配置与 Session 内可变资源状态分开，使每个 Scheduler 从同一 Spec 创建时都拥有独立、干净的 Idle 状态。

## 设计

- `NodeSpec` 保存 Name、Capacity、Labels。
- `NodeState` 保存 Spec 字段以及 Session 内的 Idle。
- `Scheduler.NewFromSpecs` 创建独立 NodeState。
- 现有 `Scheduler.New([]api.Node)` 保留，转换为 NodeSpec 后委托给 `NewFromSpecs`。

## 非目标

- 不修改现有 Loader 返回类型。
- 不实现动态 Node 加入/删除事件。
- 不改变调度算法。

## 验收

- 两个 Scheduler 使用同一 Spec 时互不共享 Idle。
- Session 分配只改变 NodeState。
- 现有 CLI 与全量测试通过。
