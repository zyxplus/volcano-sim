# Session Reclaim 设计

## 目标

将 dry-run Reclaim 纳入多轮 Queue 调度会话：正常 batch 分配优先；整轮没有进展时，才为当前 DRF 选择的等待 Job 尝试一次 Reclaim。

## 行为

Reclaim 成功时，Evict 与 Allocate 同事务提交，更新 Job batch 状态并继续候选集循环。Reclaim 失败且普通分配也无进展时，Session 结束并保留原因。

## 非目标

不一次为多个等待 Job Reclaim；不接 Kubernetes API。
