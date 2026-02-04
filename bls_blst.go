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