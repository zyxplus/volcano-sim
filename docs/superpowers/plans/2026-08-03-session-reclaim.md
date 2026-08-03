# Session Reclaim Plan

**目标：** 正常 batch 无进展时自动尝试一次 Reclaim，让 Queue 会话继续调度。

1. 写无空闲 GPU、但高 priority 等待 Job 可回收低 priority victim 的失败测试。
2. 添加会话入口：普通 RunWithQueues 无 allocation 时调用 RunWithReclaim。
3. 成功时合并 Evictions/Allocations 并保留 Job batch 状态；失败时保留不可调度原因并结束。
4. 全量测试、提交、合并。
