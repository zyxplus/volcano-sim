# Victim Priority 设计

Job 新增整数 `priority`，默认 0。Reclaim 在候选 victim 中按 Job priority 升序、JobName 升序、TaskIndex 升序选择。该阶段只影响 victim 顺序；不要求等待 Job 的 priority 高于 victim，也不改变 Queue overused/reclaimable 前置条件。

验收：两个同样合格的 victim 中，priority 10 的 Task 在 priority 100 的 Task 之前被 Evict。
