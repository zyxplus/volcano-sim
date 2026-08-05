# SessionController.Commit 设计

## 目标

把 Scheduler 输出的 AllocationPlan 应用到 SessionController 的 RunningTask 快照，形成 Run → Commit → Run 的闭环。

## 规则

- Allocation 根据 Job Spec 生成 RunningTask，继承 Job priority、Queue 和 Request。
- Eviction 按 JobName、TaskIndex、NodeName 移除 RunningTask。
- Commit 只更新内存快照，不执行 Kubernetes Bind/Evict。
- 下一次 Run 从更新后的 RunningTask 重建 Job、Queue 和 Node 状态。

## 非目标

- 不处理外部执行失败或幂等重试。
- 不持久化事件或运行态。
- 不改变 Scheduler 的调度算法。

## 验收

- Run 产生 Allocation 后 Commit，下一次 Run 不重复分配同一 Task。
- Commit Eviction 后对应 RunningTask 被移除，资源可再次调度。
- 全量测试通过。
