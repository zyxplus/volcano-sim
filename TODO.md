# 后续迭代

## Teardown Analysis 记录（2026-08-05）

### 已确认并已修复

- Queue 不存在时返回 `queue not found`，不再静默忽略 Job。
- Queue 重名和 Job 重名在 Loader 阶段拒绝，Scheduler 保留运行时兜底。
- Session 普通调度无进展后，会按 DRF 顺序逐个尝试 Reclaim Candidate。
- Reclaim 只计算 `max(0, MinAvailable-ScheduledReplicas)` 的剩余 Gang 需求。
- Reclaim 失败会回滚 Node Idle 和源 Queue Allocated。
- 队列排序后，回滚按 Queue 名称恢复，避免旧下标污染错误队列。

### 高优先级待改

1. **Queue Proportion-aware 调度**
   - 当前 Queue 之间只按 Weight 排序。
   - 目标：使用 deserved / allocated / weight 参与 Queue 选择。
   - Queue 内部继续使用 Job DominantShare（DRF）。

2. **Victim Job 的 MinAvailable 约束**
   - 当前 Reclaim 检查了 victim Queue 和 priority，但没有检查驱逐后 victim Job 是否破坏自身 Gang 要求。
   - 目标：增加 victim Job 状态索引，拒绝导致其低于 MinAvailable 的驱逐，或明确采用整 Job 驱逐策略。

### 中优先级待改

3. **NodeSpec / NodeState 分离**
   - 当前 Loader 初始化 `Node.Idle`，把运行时状态混入配置对象。
   - 目标：Loader 只产生不可变 NodeSpec；Session 初始化 NodeState（Idle、Used）。

4. **统一 ValidationResult / 错误分级**
   - 当前 Loader 返回 fatal error，Scheduler 对未知 Queue 等问题返回 Unschedulable。
   - 目标：明确哪些错误阻止整个场景，哪些只跳过单个 Job。

5. **动态控制循环**
   - 当前项目模拟单次快照内的多轮 Session，不模拟长期运行的 Volcano Controller。
   - 未来再支持 Job/Node 动态加入、删除和多 Session 事件循环。

### 暂缓

- Statement / Journal 独立抽象：当前局部 `trialAllocation` 已能满足模拟器。
- RoundPlan / SessionPlan 扩展：当前主要是结果组织，不是调度核心。
- Kubernetes Bind / Evict 执行器：等调度决策语义稳定后再接入。

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
