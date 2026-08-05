# RemoveNode RunningTask 保护设计

## 目标

避免删除仍承载运行任务的节点，防止 SessionController 产生悬空 RunningTask。

## 规则

- RemoveNode 前检查当前 RunningTask 的 NodeName。
- 仍有任务时返回错误，不修改节点或任务快照。
- 任务已通过 UpdateRunningTasks 清除后，允许删除节点。

## 非目标

- 不自动驱逐或迁移节点任务。
- 不执行 Kubernetes Node drain。

## 验收

- 有任务节点删除失败且状态不变。
- 空闲节点可以删除。
- 清除任务后节点可以删除。
