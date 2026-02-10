package main

import (
	"crypto/rand"
	"fmt"
)

// 默认 BLS 接口（Stub），用于无 blst 环境快速测试
// 这是一个最小化的接口定义，实际项目中会使用真实的 BLS 实现（例如 blst）。
type BLS interface {
	// 对 message 签名，返回签名字节序列或错误
	Sign(message []byte) ([]byte, error)
	// 聚合多个签名，返回聚合后的签名或错误
	AggregateSignatures(sigs [][]byte) ([]byte, error)
	// 验证聚合签名：给定公钥列表、消息和聚合签名，返回验证结果或错误
	VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error)
	// 返回该 BLS 实例对应的公钥（序列化字节）
	PublicKey() []byte
}

// SimpleBLSStub 是一个非常简单的 BLS 假实现（stub），仅用于本地测试或在没有 blst 库时使用。
// 它不会提供真实的密码学安全性，仅用于模拟签名、聚合与验证的流程。
type SimpleBLSStub struct {
	id int // 节点编号，用于生成可读的伪签名/公钥
}

// 构造函数：创建并返回一个新的 SimpleBLSStub 实例
func NewSimpleBLSStub(id int) *SimpleBLSStub {
	return &SimpleBLSStub{id: id}
}

// Sign 生成一个伪签名：以文本前缀 + 随机字节构成
// 注意：这里使用 crypto/rand 仅为了生成随机内容以区分签名，但这不是实际的 BLS 签名算法。
// 在生产环境中应使用真实的 BLS 签名实现。
func (s *SimpleBLSStub) Sign(message []byte) ([]byte, error) {
	// 生成 8 字节随机数作为签名主体的一部分（用于区分签名）
	b := make([]byte, 8)
	_, _ = rand.Read(b) // 忽略错误：仅用于 stub；真实实现应处理错误
	// 返回形如 "SIG-node-01-<随机字节>" 的字节切片
	return append([]byte(fmt.Sprintf("SIG-node-%02d-", s.id)), b...), nil
}

// AggregateSignatures 将多个签名拼接并加上前缀 "AGG:" 以表示聚合签名
// 该聚合方式仅为演示，不是任何标准的聚合签名格式。
func (s *SimpleBLSStub) AggregateSignatures(sigs [][]byte) ([]byte, error) {
	agg := []byte("AGG:") // 聚合签名前缀，便于 VerifyAggregate 简单判断
	for _, sg := range sigs {
		// 逐个附加签名字节
		agg = append(agg, sg...)
		// 用分号分隔各个签名，便于人工查看
		agg = append(agg, ';')
	}
	return agg, nil
}

// VerifyAggregate 对聚合签名进行“伪验证”：仅检查聚合签名是否以 "AGG:" 为前缀
// 这显然不是安全的验证，仅用于测试流程和逻辑（例如 PBFT 协议流程）。
func (s *SimpleBLSStub) VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error) {
	// Stub 验证：只判断聚合签名是否包含我们设定的前缀
	if len(aggSig) >= 4 && string(aggSig[:4]) == "AGG:" {
		return true, nil
	}
	// 非预期格式则返回失败
	return false, nil
}

// PublicKey 返回该节点的伪公钥字符串，格式为 "PK-node-01"
// 仅用于演示和在测试中作为公钥占位符。
func (s *SimpleBLSStub) PublicKey() []byte {
	return []byte(fmt.Sprintf("PK-node-%02d", s.id))
}

// NewBlstBLS 是一个仅在非 blst 构建时使用的桩函数
// 当不使用 blst 标签编译时，此函数简单地返回 SimpleBLSStub
// 这样可以让代码在没有 blst 库时也能编译通过
func NewBlstBLS(id int) BLS {
	return NewSimpleBLSStub(id)
}
