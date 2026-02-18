# 改进 PBFT 模拟器（含奖励函数、吞吐量分层、BLS 聚合接口）

说明
- 这是一个用 Go 编写的原型模拟器，用于演示对 PBFT 的三项改进：
  1. 奖励函数 m：节点按参与共识的成功/失败调整 m；当 m > mmax 则重置为 m0；当 m < mmin 则判定为恶意并排除参与共识与广播。
  2. 吞吐量分层：根据节点吞吐量（例如 TPS）将节点分为 High / Normal / Low，以便根据交易需求匹配计算能力。
  3. BLS 聚合签名：使用 blst（supranational/blst）进行签名聚合与聚合验证，显著降低消息验证开销与签名流量。

仓库结构（示例）
- main.go — 启动模拟器
- node.go — 节点与奖励/分层逻辑
- pbft.go — 简化 PBFT 流程（PRE-PREPARE / PREPARE / COMMIT）
- bls_stub.go — 默认的非安全 Stub（用于无 blst 情况快速运行与开发）
- bls_blst.go — blst 的实现（通过 build tag: `blst` 编译）
- pbft_test.go — 基本单元测试
- config.go — 全局参数（m0/mmax/mmin 等）

先决条件
- Go 1.20+
- 若使用 blst（推荐用于真实性能测试与生产），需要系统 C 编译链（gcc/clang、make、cmake）
- 若不使用 blst：可以直接运行（默认使用 SimpleBLSStub）

如何替换 BLS Stub 为 blst（明确步骤）
本项目用 build tag `blst` 选择 blst 实现。替换步骤如下（以 Ubuntu/macOS 为例）：

1. 安装系统依赖（Ubuntu 示例）
   - sudo apt-get update
   - sudo apt-get install -y build-essential cmake git

2. 获取并构建 blst（可选，某些 Go 绑定会自动构建）
   - git clone https://github.com/supranational/blst
   - cd blst
   - make
   - sudo make install     # 可选：把 lib/headers 放到系统路径，或记住路径以供 CGO 使用

3. 在你的模块中获取 Go 绑定
   - go get github.com/supranational/blst/bindings/go

4. 如需额外指定 blst 的 include/lib 路径（若未安装到系统路径）
   - export CGO_CFLAGS="-I/path/to/blst/include"
   - export CGO_LDFLAGS="-L/path/to/blst/lib -lblst"

5. 使用 build tag 编译/测试/运行（启用 blst 实现）
   - go test -tags blst ./...
   - go build -tags blst -o sim_blst ./...
   - ./sim_blst

提示
- 本仓库中 `bls_blst.go` 仅在 `-tags blst` 时参与编译；否则默认使用 `bls_stub.go`。
- 生产系统不要使用仓库中的 Stub；请务必使用 blst 或其他成熟实现，并做好私钥安全管理（HSM/秘钥库）。

运行（快速）— 不启用 blst（默认）
1. 初始化模块（若尚未）
   - go mod init improved-pbft
2. 运行
   - go run ./...
3. 运行测试
   - go test ./...

运行（启用 blst）
1. 完成上面的 blst 安装步骤
2. 运行
   - go test -tags blst ./...
   - go build -tags blst -o sim_blst ./...
   - ./sim_blst

三项改进的具体测试方法（可重复、可量化）

下面给出每项改进的测试设计、实现要点与如何量化结果。

A. 奖励函数 m（正确性与收敛性测试）
目标：验证奖励机制按设计更新 m 值，能惩罚持续作恶节点并最终将其排除；并且 m>mmax 时可以防止单节点长期集中化（reset）。

测试步骤
1. 准备场景：
   - 启动 N 节点（例如 N=7），指定 K 个固定恶意节点（例如 K=2），余下为正常节点。
   - 配置 InitialM、MMax、MMin（例如 5、10、0）。

2. 执行：
   - 运行 T 轮共识（例如 T=50~200），记录每轮结束后所有节点的 m 值与 active 状态。

3. 验证点（断言/手动检查）：
   - 恶意节点的 m 随时间应下降并在 m < mmin 时被标记 inactive（active == false）。
   - 好节点在多次成功参与后 m 会增加；当某节点 m > mmax 时会被重置到 m0（观察到重置事件）。
   - 记录并计算“被排除的节点数 vs 恶意节点数”的一致性（应该至少覆盖所有持续恶意节点）。

衡量指标与输出
- 曲线：每个节点 m 值随轮次变化图（可输出为 CSV）
- 总体被排除节点数与定位时间（从开始到 m<mmin 的轮数）
- 共识成功率随时间（是否因删除恶意节点提高）

B. 吞吐量分层（正确性与匹配性测试）
目标：验证分层算法（ComputeTiers）确实将前 30% 标记为 High，后 30% 标记为 Low，并验证在基于需求分配 leader/任务时高层节点更多被选中参与高吞吐需求交易。

测试步骤
1. 准备场景：
   - 构造 N 个节点并赋予人工设定吞吐量值（可分为明显三段：Top 30% 高吞吐，Mid 40% 中等，Bottom 30% 低吞吐）。

2. 执行：
   - 调用 sim.ComputeTiers()，并输出每个节点的 Tier 字段。
   - 针对一类高吞吐需求的模拟请求（例如 request 模拟带大并发），运行多轮，观察被选为 leader 的节点分布。

3. 验证点：
   - Tier 分配应严格按吞吐量排序切分（Top/Bottom 大小符合比例）。
   - 对于高吞吐需求的多轮测试，leader 被选中时 High-tier 节点的占比应明显高于随机基线（统计比率）。

衡量指标与输出
- Tier assignment table（节点 => throughput => Tier）
- Leader selection频率按Tier分布（High: %, Normal: %, Low: %）
- 平均请求处理延迟（高吞吐请求在 High 节点参与时是否降低）

C. BLS 聚合签名（使用 blst）的正确性与性能测试
目标：验证使用 blst 的聚合签名在功能上正确并在性能/带宽上比逐一验证优越。

功能性测试（正确性）
1. 单独签名与聚合签名：
   - 让 M 个节点对同一消息签名（单签名序列）。
   - 使用 leader 的 AggregateSignatures 将这些签名聚合。
   - 使用 VerifyAggregate 验证聚合签名；应返回 true。

2. 异常路径：
   - 含有一个或多个错误签名时，聚合验证应失败或通过适当手段检测到错误（具体实现依赖聚合方式：多消息 / 多公钥场景）。

运行指令
- go test -tags blst ./...  # 包含 blst 的单元/集成测试

性能测试（量化）
1. 对比实验设计
   - 场景 A（baseline）：每个签名由 leader 单独验证（模拟不聚合，leader 验证 n 次）。
   - 场景 B（blst 聚合）：节点签名聚合后 leader 做一次聚合验证。

2. 测量项
   - 每轮 PREPARE 阶段：leader 验证总耗时（ms）
   - 每轮消息大小（用于模拟网络，统计签名字节总量）
   - CPU 使用率（leader 节点）与每轮延迟
   - 吞吐（requests/sec）在相同条件下的差异

3. 实验步骤
   - 对两个场景在相同节点数与相同消息负载下运行多次（例如 30~100 轮），收集统计数据。
   - 使用 Go 的 time/pprof 或简单的 time.Now() 记录关键步骤耗时（签名生成、聚合、验证）。

期望与判断
- 聚合验证场景应显著降低 leader 的验签总耗时，且网络上传输的签名字节总量应降至常规签名方式的 O(1)（聚合签名大小）而非 O(n)。
- 延迟与吞吐：启用 blst 后，在签名验证成为瓶颈的场景应看到吞吐提升或平均延迟下降。

测试用例/脚本建议
- 提供两个脚本：
  - correctness_test.sh：运行多轮聚合正确性测试（使用 -tags blst）
  - perf_test.sh：分别用 stub 和 blst 构建并运行同一负载，收集 CSV 格式耗时/bytes，以便绘图对比

示例命令（blst）
- go test -tags blst ./...                             # 单元/集成测试
- go test -tags blst -run TestBLSAggregation ./...     # 如有单独 BLS 测试
- go build -tags blst -o sim_blst ./...
- ./sim_blst > run_blst.log 2>&1

对比命令（stub）
- go test ./...
- go build -o sim_stub ./...
- ./sim_stub > run_stub.log 2>&1

如何收集/导出数据
- 在 pbft.go / main.go 的关键点插入日志计时（例如 PREPARE start/end, AGGREGATE start/end, VERIFY start/end），并把结果写 CSV（轮次、step、duration_ms、sig_bytes、commit_sigs_count）。
- 使用 Linux 工具（time, top, sar）或 go pprof 对 CPU 做采样分析。

Fabric 集成思路（摘要）
- 可将改进 PBFT 封装为 Fabric ordering service 的一个实现或插件：
  - Leader 选举：基于 m 值与 Tier 策略调整 ordering 节点优先级。
  - 签名聚合：在 ordering 节点之间使用 BLS 聚合签名替代逐个 ECDSA 验证/签名（需统一消息 hash_to_curve/DST）。
  - 恶意节点排除：当节点 m<mmin 时通过 channel reconfiguration（或管理员 API）将其移出 consenter 列表。
- 吞吐量统计来源：ordering 节点的监控（Prometheus metrics、日志）用于周期性更新吞吐量值并重新分层。

排查要点（常见问题）
- 编译错误（找不到 blst 绑定或 CGO 链接错误）
  - 检查 CGO_CFLAGS/CGO_LDFLAGS 是否指向 blst 的 include/lib
  - 确认 `go get github.com/supranational/blst/bindings/go` 成功
- 聚合验证失败
  - 确认所有节点使用相同的 BLS 参数（G1/G2 约定、DST、hash_to_curve 版本）
  - 确认签名/公钥序列化方法一致（压缩/非压缩）
- 结果偏差（预期中的提升不明显）
  - 检查是否真正把 leader 的验证从 O(n) 变为 O(1)（有无错误地多次反序列化/验证）
  - 检查网络/IO 或其他组件是否为瓶颈（而非签名验证）

后续建议
- 把 BLS 私钥管理接入安全模块（HSM 或 Vault），不要在进程内裸生成并保存长期密钥。
- 在真实网络/容器环境中做 end-to-end 压测（不同节点延迟/带宽、节点故障/重启、恶意节点行为）。
- 在集成 Fabric 前，先在小规模独立 ordering-service 原型中完成验证与 reconfiguration 流程设计。

许可证与安全提示
- 本模拟器原型仅用于研究/实验；不要把 Stub 用于生产。
- 使用 blst 时请遵循其 Apache-2.0 许可证与绑定库的许可证条款。
- 生产部署前请完成安全审计，特别是密钥管理与消息序列化/哈希细节。

联系方式与下一步
- 如果你需要，我可以：
  - 提供针对 Ubuntu 22.04 / macOS 的一键安装脚本（包含 blst、CGO 环境变量设置与 build/test 指令）。
  - 为 BLS 聚合测试补充具体的 benchmark 脚本（生成 CSV，绘图脚本）。
  - 将 PBFT 改进封装为 Fabric ordering-plugin 的原型代码片段并说明如何做 channel reconfiguration。

感谢阅读——请告诉我希望我接着生成哪部分（安装脚本、bench 脚本、或 Fabric 插件草案）。  

启动前端前需要先启动/front中的后端服务；
