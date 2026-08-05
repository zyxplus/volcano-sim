# SessionController.Commit 原子性设计

## 目标

Commit 对一个 AllocationPlan 采用全有或全无语义，避免 Eviction 已生效而后续 Allocation 校验失败造成半提交。

## 规则

- 先复制 RunningTask 快照。
- 在副本上验证并应用所有 Eviction 和 Allocation。
- 任意一项非法时返回错误，正式快照不变。
- 全部合法后一次性替换正式快照。

## 非目标

- 不处理外部 Kubernetes 执行失败。
- 不引入数据库事务或持久化日志。

## 验收

- 混合计划中任一项非法时，所有变更都不生效。
- 合法计划仍能完整提交。
- 全量测试通过。
