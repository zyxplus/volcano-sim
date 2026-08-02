# Volcano-Sim Milestone 1 设计

## 目标

实现一个纯内存、可由 YAML 驱动的 Go 调度模拟器。它演示两项 Volcano Lite 核心语义：

1. 按 DRF（Dominant Resource Fairness）排序 Job。
2. 为 Job 进行 Gang 试分配；只有达到 `minAvailable` 才提交，失败则完整回滚。

本阶段不接入 Kubernetes，不实现 Queue、拓扑、抢占或回收。

## 输入与输出

输入由 `testdata` 中的 `nodes.yaml` 与 `jobs.yaml` 提供。资源使用通用向量
`map[string]float64` 表示，支持任意维度，例如 `cpu`、`memory`、`gpu`。

输出是纯数据的 `AllocationPlan`：已提交的 Task 到 Node 分配，以及每个未调度 Job 的原因。

## 包边界

```text
cmd/volcano-sim/       命令行入口；加载数据、运行一次调度、打印计划
pkg/api/               Resource、Node、Job、Task、AllocationPlan
pkg/loader/            YAML 解码与输入校验；不含调度策略
pkg/scheduler/         DRF 排序、Gang 试分配和回滚
pkg/scheduler/*_test.go 端到端和单元测试
testdata/              人可读的调度场景
```

`api` 不依赖其余包；`loader` 和 `scheduler` 只依赖 `api`；`cmd` 负责组合它们。

## 调度流程

1. Loader 读取并校验 YAML，构造 Node 和 Job。
2. Scheduler 汇总所有 Node 的容量，计算每个 Job 当前已分配资源的 dominant share。
3. Scheduler 按 dominant share 从小到大排序 Job；相同份额以 Job 名称稳定排序。
4. 对每个 Job 创建一次临时试分配。依次为待调度 Task 选择可容纳其请求的 Node。
5. 试分配数达到 `minAvailable` 时，提交所有临时占用并写入计划；否则恢复每个受影响 Node 的空闲资源，并记录不可调度原因。

首版假设同一 Job 的所有 Task 资源请求一致。Node 选择采用稳定的名称顺序；这使 Gang 和 DRF 的行为可预测且便于测试。

## 正确性与失败处理

- `Resource` 运算返回新值，避免向量别名导致共享状态被悄然修改。
- 每次临时占用都记录对应 Node 与资源；回滚按记录逐项恢复，确保无部分 Gang 分配遗留。
- 非法 YAML、缺失名称、非正资源请求或 `minAvailable` 超过副本数会返回带上下文的错误，绝不 panic。
- 合法但资源不足的 Job 不是程序错误：它保留为未调度并输出 `insufficient resources for minAvailable`。

## 验收测试

1. Resource：加减、比较、克隆与 dominant share 支持任意资源键。
2. Gang：两个均要求 4 个 GPU Task 的 Job、但集群只剩 4 GPU 时，不得留下任一 Job 的部分分配。
3. DRF：CPU-heavy 与 GPU-heavy Job 按较小 dominant share 优先。
4. Loader：从 YAML 构造对象，并拒绝无效输入。
5. CLI：指定场景后输出可读、确定的调度计划。

## 后续阶段

Milestone 2 加入 Job 级 Fabric 预选。Milestone 3 引入 Queue、Proportion 与 Reclaim；届时再抽取 Session/Plugin 框架，避免在本阶段过度设计。
