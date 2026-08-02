# 后续迭代

## 动态 DRF 重排

当前每个 Job 在一个调度轮中只尝试一次 Gang 分配；成功后即从候选集中移除。因此在成功后更新它的 dominant share 并重排，不会影响本轮的后续选择。

在实现动态 DRF 前，需要先支持一个 Job 跨多个调度轮持续存在，例如：

- 为 Job 增加 RemainingReplicas / 已提交 replica 状态；
- 每轮只为 Job 提交一个可配置大小的 Gang batch；
- 已达到 minAvailable 但尚未完成的 Job 留在候选集；
- 每次 batch 提交后更新 Job.Allocated，再对同一 Queue 的剩余 Job 重新执行 DRF 排序。

验收场景：两个 Job 的资源需求不同；第一个 Job 获得一个 batch 后 dominant share 上升，第二个 Job 在下一轮必须被优先选择。
