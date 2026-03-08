package node

import (
	"crypto/rand"
	"fmt"
)

// ======================= 【高亮-2026-03-08】新增：node 包自带 BLS 接口与 Stub，实现解耦（node 包不再依赖 pbft1 包） =======================

// BLS：node 层的通用 BLS 接口（算法层可复用）
type BLS interface {
	Sign(message []byte) ([]byte, error)
	AggregateSignatures(sigs [][]byte) ([]byte, error)
	VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error)
	PublicKey() []byte
}

// SimpleBLSStub：非安全 stub，仅用于本地仿真/无 blst 环境
type SimpleBLSStub struct {
	id int
}

func NewSimpleBLSStub(id int) *SimpleBLSStub {
	return &SimpleBLSStub{id: id}
}

func (s *SimpleBLSStub) Sign(message []byte) ([]byte, error) {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return append([]byte(fmt.Sprintf("SIG-node-%02d-", s.id)), b...), nil
}

func (s *SimpleBLSStub) AggregateSignatures(sigs [][]byte) ([]byte, error) {
	agg := []byte("AGG:")
	for _, sg := range sigs {
		agg = append(agg, sg...)
		agg = append(agg, ';')
	}
	return agg, nil
}

func (s *SimpleBLSStub) VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error) {
	if len(aggSig) >= 4 && string(aggSig[:4]) == "AGG:" {
		return true, nil
	}
	return false, nil
}

func (s *SimpleBLSStub) PublicKey() []byte {
	return []byte(fmt.Sprintf("PK-node-%02d", s.id))
}
