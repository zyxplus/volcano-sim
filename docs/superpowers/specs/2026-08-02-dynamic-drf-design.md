# Dynamic DRF 设计

每个 Queue 在一个调度轮内不再只排序一次。每成功提交一个 Gang Job 后，调度器把该 Job journal 的资源总量写入 Job.Allocated，再从同一 Queue 的剩余 Job 中重新选择 dominant share 最小者。Gang 失败不改变 Job.Allocated。Queue weight 的排序不变。

测试场景：两个 Job 初始 dominant share 相同；第一个提交 GPU 后，它的 share 上升，第二个 Job 必须在下一次被选择。
