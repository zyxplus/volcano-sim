# Full Session Loop Plan

**目标：** 在一次 Session 内持续调度 batch；有进展继续、无进展退出，并汇总纯决策输出。

1. 写两个可调度 Job 在一次 RunSession 中都得到 Allocation 的失败测试。
2. 将 RunSession 改为 pending 循环，合并各轮 AllocationPlan。
3. 用“本轮无 Allocation”作为退出条件，保留不可调度原因。
4. 全量测试、提交、合并。
