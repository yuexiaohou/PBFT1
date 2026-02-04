//go:build blst
// +build blst

package main

import (
    "crypto/rand"
    "fmt"
    blst "github.com/supranational/blst/bindings/go"
)

// 每个节点持有的BLS密钥对
type BlstBLS struct {
	sk *blst.SecretKey
	pk *blst.P1Affine
	id int
}

// 新建 BLS 密钥对
func NewBlstBLS(id int) *BlstBLS {
	ikm := make([]byte, 32)
	_, _ = rand.Read(ikm)
	sk := new(blst.SecretKey)
	sk.KeyGen(ikm, nil)
	pk := new(blst.P1Affine).From(sk)
	return &BlstBLS{
		sk: sk,
		pk: pk,
		id: id,
	}
}

// 单节点签名
func (b *BlstBLS) Sign(message []byte) ([]byte, error) {
	const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
	sig := new(blst.P2Affine).Sign(b.sk, message, []byte(dst))
	return sig.Compress(), nil
}

// 聚合若干签名
func (b *BlstBLS) AggregateSignatures(sigs [][]byte) ([]byte, error) {
	if len(sigs) == 0 {
		return nil, nil
	}
	var agg blst.P2Aggregate
	sigObjs := make([]*blst.P2Affine, len(sigs))
	for i, sbytes := range sigs {
		sigObj := new(blst.P2Affine)
		if err := sigObj.Deserialize(sbytes); err != nil {
			return nil, fmt.Errorf("sig deserialize: %w", err)
		}
		sigObjs[i] = sigObj
	}
	// true 表示校验签名的有效性
	ok := agg.Aggregate(sigObjs, true)
	if !ok {
		return nil, fmt.Errorf("aggregation failed")
	}
	aggSig := agg.ToAffine()
	return aggSig.Compress(), nil
}

// 验证聚合签名
func (b *BlstBLS) VerifyAggregate(pubKeys [][]byte, message, aggSig []byte) (bool, error) {
	if aggSig == nil {
		return false, fmt.Errorf("nil aggregate signature")
	}
	var sig blst.P2Affine
	if err := sig.Deserialize(aggSig); err != nil {
		return false, fmt.Errorf("agg deserialize: %w", err)
	}
	pks := make([]*blst.P1Affine, len(pubKeys))
	for i, pkb := range pubKeys {
		var p blst.P1Affine
		if err := p.Deserialize(pkb); err != nil {
			return false, fmt.Errorf("pk deserialize: %w", err)
		}
		pks[i] = &p
	}
	const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
	// true 表示不是预哈希输入
	ok := sig.FastAggregateVerify(true, pks, message, []byte(dst))
	return ok, nil
}

// 公钥字节获取
func (b *BlstBLS) PublicKey() []byte {
	return b.pk.Compress()
}