# Full Session Loop 设计

RunSession 在 pending Job 非空时持续运行：先尝试正常 batch，若有 Allocation 则合并 plan 并继续；否则尝试 Reclaim，若有 Allocation 则合并 Evictions/Allocations 并继续；两者均无 Allocation 时结束。

目标：让多轮 batch、动态 DRF 与 Reclaim 在同一个会话中协作。退出条件是整轮无 Allocation，避免无限循环。
