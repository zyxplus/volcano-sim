# Batched Job Scheduling Plan

**目标：** 支持跨多轮的 Job batch 提交，并让动态 DRF 影响下一批选择。

1. 给 Job 增加 BatchSize/ScheduledReplicas，测试 YAML 默认与状态更新。
2. 将一次性 Gang 试分配提取为单 batch 事务：首次至少 minAvailable，后续上限 batchSize。
3. 将 Queue 内循环改为“选择一个未完成 Job → 提交一 batch → 重新 DRF 排序”。
4. 添加 A/B 交替 batch 的失败测试，完整测试、提交、合并。
