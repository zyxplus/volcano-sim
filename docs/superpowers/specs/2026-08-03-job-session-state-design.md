# Job Session State 设计

Scheduler 的批调度需要回写 Job.ScheduledReplicas 与 Job.Allocated，因此 `Run`/`RunWithQueues` 的 Job 输入改为 `[]*api.Job`。成功 journal commit 后增加两项状态；失败回滚不改 Job 状态。

目标：为后续多轮 batch 与动态 DRF 建立共享可变 Job 会话状态。
