package main

import (
	"crypto/rand"
	"fmt"
)

// 默认 BLS 接口（Stub），用于无 blst 环境快速测试
type BLS interface {
	Sign(message []byte) ([]byte, error)
	AggregateSignatures(sigs [][]byte) ([]byte, error)
	VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error)
	PublicKey() []byte
}

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
	// Stub 验证：只判断前缀
	if len(aggSig) >= 4 && string(aggSig[:4]) == "AGG:" {
		return true, nil
	}
	return false, nil
}

func (s *SimpleBLSStub) PublicKey() []byte {
	return []byte(fmt.Sprintf("PK-node-%02d", s.id))
}
