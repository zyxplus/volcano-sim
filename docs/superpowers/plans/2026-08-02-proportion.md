# Proportion Calculation Plan

1. 为 Queue 增加 Guarantee、Reclaimable，并测试 YAML 读取。
2. 在新 `pkg/proportion` 中先写 32 GPU、1:3 weight、8/24 deserved 的失败测试。
3. 实现 `ComputeDeserved(queues []api.Queue, total api.Resource) map[string]api.Resource` 与 `IsOverused`；对每个资源维度按 weight 分配后 clamp 到 guarantee/capability。
4. 运行 `go test ./...`，提交并合并。
