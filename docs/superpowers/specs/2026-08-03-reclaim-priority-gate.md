# Reclaim Priority Gate 设计

Reclaim 仅在等待 Job 的 priority 严格大于候选 RunningTask priority 时选择该 victim。Queue overused/reclaimable 与最少资源回收条件保持不变。

验收：priority 10 的等待 Job 不得驱逐 priority 100 victim；priority 100 的等待 Job 可以驱逐 priority 10 victim。
