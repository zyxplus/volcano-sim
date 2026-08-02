# Fabric Gang Threshold 设计

## 目标

让 Required Fabric 的可行性门槛与 Gang 提交门槛保持一致。

## 行为

`selectCandidateNodes` 将单个 Fabric 所需资源从 `Replicas × Request` 改为 `MinAvailable × Request`。因此一个 `replicas: 8`、`minAvailable: 4`、每副本 1 GPU 的 Job，只要单个匹配 Fabric 有 4 个空闲 GPU，就可以开始试分配并提交 4 个 Task。

Fabric 仍必须在 journal 建立前被选定；journal 仍会在无法达到 `minAvailable` 时回滚所有临时资源占用。

## 测试

添加一个 Required Fabric 测试：单 Fabric 有 4 GPU，Job 有 8 个副本且 `minAvailable` 为 4。计划应包含 4 个 allocation，且均位于该 Fabric。
