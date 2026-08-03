# Victim Priority Plan

**目标：** Reclaim 优先驱逐低 priority Task，减少对关键 Job 的影响。

1. 为 Job 增加 priority YAML 字段并测试默认值。
2. 为 RunningTask 带入 Priority。
3. 写两个 victim priority 10/100 的失败测试，期望先 Evict 10。
4. 在 RunWithReclaim 开始前稳定排序 victims；全量测试、提交、合并。
