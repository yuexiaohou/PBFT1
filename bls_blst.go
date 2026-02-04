//go:build blst
// +build blst

package main

import (
    "crypto/rand"
    "fmt"
    blst "github.com/supranational/blst/bindings/go"
)

type BlstBLS struct {
    sk *blst.SecretKey
    pk *blst.P1Affine
    id int
}

// 新建密钥对
func NewBlstBLS(id int) *BlstBLS {
    ikm := make([]byte, 32)
    if _, err := rand.Read(ikm); err != nil {
        panic(err)
    }
    sk := blst.KeyGen(ikm)
    pk := new(blst.P1Affine).From(sk)
    return &BlstBLS{sk: sk, pk: pk, id: id}
}

func (b *BlstBLS) Sign(message []byte) ([]byte, error) {
    const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
    sig := new(blst.P2Affine).Sign(b.sk, message, []byte(dst))
    return sig.Compress(), nil
}

func (b *BlstBLS) PublicKey() []byte {
    return b.pk.Compress()
}

// 聚合签名
func (b *BlstBLS) AggregateSignatures(sigs [][]byte) ([]byte, error) {
    if len(sigs) == 0 {
        return nil, nil
    }
    var agg blst.P2Aggregate
    sigObjs := make([]*blst.P2Affine, len(sigs))
    for i, sbytes := range sigs {
        sigObj := new(blst.P2Affine)
        if err := sigObj.Deserialize(sbytes); err != nil {
            return nil, err
        }
        sigObjs[i] = sigObj
    }
    agg.Aggregate(sigObjs, true)
    aggSig := agg.ToAffine()
    return aggSig.Compress(), nil
}

// 聚合签名验证
func (b *BlstBLS) VerifyAggregate(pubKeys [][]byte, message, aggSig []byte) (bool, error) {
    if aggSig == nil {
        return false, nil
    }
    sig := new(blst.P2Affine).Deserialize(aggSig)
    pks := make([]*blst.P1Affine, len(pubKeys))
    for i, pkb := range pubKeys {
        pks[i] = new(blst.P1Affine).Deserialize(pkb)
    }
    const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
    ok := sig.FastAggregateVerify(true, pks, message, []byte(dst))
    return ok, nil
}