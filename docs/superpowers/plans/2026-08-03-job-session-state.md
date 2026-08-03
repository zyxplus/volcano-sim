# Job Session State Plan

**目标：** 用 `[]*api.Job` 保持跨 batch 的可变会话状态。

1. 将 Scheduler Run/RunWithQueues/排序与测试输入迁移到 Job 指针。
2. 成功 journal commit 后更新 ScheduledReplicas 和 Allocated；失败不更新。
3. 更新 Loader/CLI 构造的 Job 指针，运行全量测试并提交。
