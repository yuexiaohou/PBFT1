+-------------------+
|                   |
|   PBFTSimulator   |
|                   |
+--------+----------+
|
| 含有 []*Node
v
+----[循环]: for round in Rounds ---+
|                                   |
| 1. SelectLeader()                 |
|    |                              |
|    v                              |
| 2. ComputeTiers()                 |
|    +                              |
| 3. request := new message         |
|    +                              |
| 4. for each node:                 |
|      - Sign(request)              |
|      - 收集签名                   |
|    +                              |
| 5. leader:                        |
|      - 聚合签名 AggregateSignatures|
|      - VerifyAggregate            |
|      - 如果通过, 记成功信息         |
[...下一轮...]

+-------------------+
| Individual Node   |
+-------------------+
| id                |
| isMalicious       |
| throughput (tp)   |
| tier              |
| active            |
| bls (BLS接口实现) |
+-------------------+
|
|---> bls.Sign()             // 签名
|---> bls.AggregateSignatures()
|---> bls.VerifyAggregate()
|---> bls.PublicKey()

+-------------------+
|   BLS 接口        |（抽象接口+blst具体实现）
+-------------------+

[备注]
- 各节点有不同层级、恶意属性、througput更新，每轮可能状态变化
- 通过BLS签名聚合提升性能
- leader由PBFTSimulator动态选取