# Session Reclaim Candidate 设计

## 目标

当普通调度无进展时，Session 按 DRF 顺序逐个尝试 pending Job 的 reclaim；某个 Job 成功后进入下一轮，全部失败才结束。

## 规则

- 每个 Job 在一次 reclaim 阶段最多尝试一次。
- 尝试顺序由 `OrderJobs` 决定。
- 成功的 reclaim 消耗对应 victims，并由下一轮重新构造 pending 集合。
- 失败的 candidate 不产生 Eviction，也不改变节点或队列状态。

## 非目标

- 不改变单个 Job 的 victim 排序和 priority 门槛。
- 不在一次 reclaim 中同时满足多个 Job。
- 不改变普通调度顺序。

## 验收

- 第一个 candidate 失败、第二个 candidate 成功时，第二个 Job 获得资源。
- 全部 candidate 失败时 Session 正常退出。
- 不重复驱逐同一 victim。
