# Queue Admission 设计

## 目标

为 Volcano-Sim 加入单层 Queue，使 Job 能按 Queue 权重调度，并受到 Queue capability 的资源上限约束。

## 输入

新增 `queues.yaml`：

```yaml
queues:
  - name: inference
    weight: 3
    capability: {gpu: 32}
```

Job 新增可选 `queue`。缺省 Job 归入名称为 `default` 的队列；Loader 在未提供 queues.yaml 或其中缺少 default 时创建无限 capability、weight 为 1 的 default Queue。

## 调度行为

1. Queue 按 `weight` 降序、Name 升序排序。
2. 在每个 Queue 内，复用现有 DRF 顺序。
3. Job 的每项 journal 试分配都先检查 `Queue.Allocated + journalResource + Request` 是否 Fits Queue.Capability。
4. 若 capability 不足，终止该 Job 的试分配、回滚 journal，并记录 `queue capability exceeded`。
5. Gang 成功时，journal 资源总量加入 Queue.Allocated；Gang 失败或 capability 失败不得改变 Queue.Allocated。

## 范围

本阶段不包含 guarantee、借用、Proportion、reclaim 或多级 Queue。Queue capability 是硬上限；资源空闲也不能突破。

## 测试

1. 更高 weight Queue 的 Job 先得到资源。
2. capability 为 4 GPU 的 Queue 拒绝 minAvailable 为 8、每 Task 1 GPU 的 Gang。
3. 相同 Queue 内仍由 DRF 排序。
4. 未填写 queue 的 Job 被分到 default。
