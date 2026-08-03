# 后续迭代

## 动态 DRF 重排

当前每个 Job 在一个调度轮中只尝试一次 Gang 分配；成功后即从候选集中移除。因此在成功后更新它的 dominant share 并重排，不会影响本轮的后续选择。

在实现动态 DRF 前，需要先支持一个 Job 跨多个调度轮持续存在，例如：

- 为 Job 增加 RemainingReplicas / 已提交 replica 状态；
- 每轮只为 Job 提交一个可配置大小的 Gang batch；
- 已达到 minAvailable 但尚未完成的 Job 留在候选集；
- 每次 batch 提交后更新 Job.Allocated，再对同一 Queue 的剩余 Job 重新执行 DRF 排序。

验收场景：两个 Job 的资源需求不同；第一个 Job 获得一个 batch 后 dominant share 上升，第二个 Job 在下一轮必须被优先选择。

## 调度决策 Trace

为 `AllocationPlan` 增加结构化或文本 Trace，记录每个 Job 的关键检查结果，例如 Queue capability、Fabric 可行性、Gang 资源不足与 Reclaim priority 门槛。

最小输出格式：`job=<name> check=<stage> result=<pass|fail|blocked>`。CLI JSON 直接返回 Trace；暂不接入 Kubernetes Event 或日志框架。

验收场景：priority 被阻止的 Reclaim 既输出 `reclaim blocked by priority`，也包含对应的 trace 记录。

## RoundPlan 与 SessionPlan 分层

当前 `AllocationPlan` 是纯内存模拟器的汇总输出，便于 CLI 与测试一次查看整个 Session 的调度决策；它不应被解释为全 Session 的原子提交。

后续拆分为 `RoundPlan`（单轮 Eviction / Allocation）与 `SessionPlan`（`[]RoundPlan` 加可选汇总视图）。接 Kubernetes 时，每个 RoundPlan 成功后即可独立执行 Bind/Evict，而不是等整个 Session 结束。
