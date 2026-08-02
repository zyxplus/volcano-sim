# Proportion 第一阶段设计

Queue 新增 `guarantee` 与 `reclaimable`。本阶段实现按 active Queue weight 分配集群资源的 deserved 计算，并将结果 clamp 到 `[guarantee, capability]`。当 Queue.Allocated 超过 deserved 时，它被识别为 overused。

不在本阶段执行 Evict 或 Reclaim；输出仅供测试和后续 victim 选择使用。

验收：32 GPU 集群中，weight 为 1 的 Queue A 与 weight 为 3 的 Queue B 分别得到 8 / 24 GPU deserved；A 已分配 20 GPU 时为 overused。
