# SessionController 并发安全设计

## 目标

保证 `Apply`、`Run`、`Commit` 对 SessionController 状态的访问和修改不会发生数据竞争。

## 规则

- Controller 使用内部互斥锁保护 nodes、jobs、queues、victims。
- 每个公开状态操作独占锁。
- `Run` 在锁内基于当前快照生成计划，调用方仍需通过 Commit 更新状态。

## 非目标

- 不实现读写锁或无锁快照。
- 不保证多个 Run 调用的业务顺序，只保证内存安全。
- 不改变单线程调度结果。

## 验收

- 并发 Run/Apply/Commit 不产生 race。
- `go test -race ./pkg/scheduler` 通过。
- 现有全量测试通过。
