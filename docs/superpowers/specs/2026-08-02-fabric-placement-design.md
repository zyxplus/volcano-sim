# Fabric Placement 设计

## 目标

为 Volcano-Sim 加入 Job 级 GPU Fabric 预选。设置 `sameFabric: Required` 的 Job 只能在一个满足 GPU 型号且拥有足够总空闲资源的 `fabric-id` 内进行 Gang 试分配。

## 输入模型

Node 新增可选 `labels`：

```yaml
labels:
  fabric-id: fabric-a
  gpu-model: c550
```

Job 新增可选 `topology`：

```yaml
topology:
  sameFabric: Required
  gpuModel: c550
```

缺少 topology 的 Job 保持现有行为。`sameFabric` 只支持 `Required`；其他值由 loader 拒绝。Required Job 的每个候选节点都必须同时有非空 `fabric-id` 和匹配的 `gpu-model`。

## 调度行为

在 `Scheduler.Run` 为 Job 建立 journal 前，先选择候选节点：

1. 若没有 Required topology，候选集为全部节点。
2. 筛选 GPU 型号匹配的节点，并按 `fabric-id` 分组。
3. 对每个 group 汇总 Node.Idle；只有其资源不小于 `Replicas × Request` 才可行。
4. 在可行 group 中选择字典序最小的 fabric-id；只使用该 group 中按 Node 名排序的节点试分配。

资源虽然会继续经由 Gang journal 试分配和回滚，但 Fabric 可行性必须在任何扣减 Idle 前完成。

## 失败与可观测性

没有单一 Fabric 满足总资源需求时，plan 为该 Job 写入 `no single fabric has enough capacity`。此失败不能产生 allocation，也不能改变任意 Node.Idle。

## 测试

1. 两个 Fabric 各有 16 GPU；一个 24 个 1-GPU Task、`minAvailable: 24` 的 Required Job 不可调度，即使集群总量为 32 GPU。
2. 单个具有足够资源的 matching Fabric 被选择，所有 allocation 都具有相同 fabric-id。
3. 无 topology 的 Job 仍可跨节点分配，保持 Milestone 1 行为。
4. Loader 拒绝 Required topology 缺少 gpuModel 或不支持的 sameFabric 值。
