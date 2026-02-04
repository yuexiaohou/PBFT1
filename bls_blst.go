//go:build blst
// +build blst

package main

import (
    "crypto/rand"
    "errors"

    blst "github.com/supranational/blst/bindings/go"
)

type BlstBLS struct {
    sk *blst.SecretKey
    pk *blst.P1Affine
    id int
}

// 新建 BLS 密钥对
func NewBlstBLS(id int) *BlstBLS {
    ikm := make([]byte, 32)
    rand.Read(ikm)
    sk := blst.KeyGen(ikm) // Note: KeyGen returns *SecretKey
    pk := new(blst.P1Affine).From(sk)
    return &BlstBLS{sk: sk, pk: pk, id: id}
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
        sigObj := new(blst.P2Affine).Deserialize(sbytes)
        if sigObj == nil {
            return nil, errors.New("aggregate: signature deserialize failed")
        }
        sigObjs[i] = sigObj
    }
    agg.Aggregate(sigObjs, true)
    aggSig := agg.ToAffine()
    return aggSig.Compress(), nil
}

// 验证聚合签名
func (b *BlstBLS) VerifyAggregate(pubKeys [][]byte, message, aggSig []byte) (bool, error) {
    if aggSig == nil {
        return false, errors.New("aggSig is nil")
    }
    sig := new(blst.P2Affine).Deserialize(aggSig)
    if sig == nil {
        return false, errors.New("aggSig deserialize failed")
    }
    pks := make([]*blst.P1Affine, len(pubKeys))
    for i, pkb := range pubKeys {
        pk := new(blst.P1Affine).Deserialize(pkb)
        if pk == nil {
            return false, errors.New("pubkey deserialize failed")
        }
        pks[i] = pk
    }
    const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
    ok := sig.FastAggregateVerify(true, pks, message, []byte(dst))
    return ok, nil
}

// 获取公钥字节
func (b *BlstBLS) PublicKey() []byte {
    return b.pk.Compress()
}