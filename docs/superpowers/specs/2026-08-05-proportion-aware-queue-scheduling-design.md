# Proportion-aware Queue 调度设计

## 目标

让 Queue 的 Weight / Guarantee / Capability 计算出的 deserved share 参与正常调度，而不是只用于 Reclaim 的 overused 判断。

## 算法

对每个有 pending Job 的 Queue 计算最大资源缺口：

`deficit = max((deserved[resource] - allocated[resource]) / total[resource])`

每次只为缺口最大的 Queue 调度一个 batch；提交后重新计算所有 Queue 的 deficit。缺口相同时按 Weight 降序、名称升序稳定排序。

## 非目标

- 不改变 Queue 内 Job 的 DRF 排序。
- 不改变 Fabric、Gang、Batch 或 Reclaim 逻辑。
- 不引入跨 Session 的公平性状态。

## 验收

- weight=1 与 weight=3 的 Queue 在 4 个同等 GPU 请求下得到约 1:3 的分配。
- Queue 达到 deserved 后，调度转向其他缺口更大的 Queue。
- 现有 Queue capability、Gang 和 Reclaim 测试通过。
