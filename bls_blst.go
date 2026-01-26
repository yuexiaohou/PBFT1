//go:build blst
// +build blst

package main

/*
blst 实现（启用 `-tags blst` 编译）

注意：
- 此文件依赖 github.com/supranational/blst/bindings/go 绑定。
- 绑定 API 会随版本变动，若出现编译错误，请参考绑定仓库 README 并按注释调整少量 API 调用。
- 生产中应从安全存储加载私钥；此处示范用随机 IKM 生成私钥仅用于演示/测试。

示例编译（启用 blst）：
  go test -tags blst ./...
  go build -tags blst -o sim_blst ./...
*/
import (
	"crypto/rand"
	"fmt"

	blst "github.com/supranational/blst/bindings/go"
)

type BlstBLS struct {
	sk *blst.SecretKey
	pk []byte
	id int
}

func NewBlstBLS(id int) *BlstBLS {
	// 仅示例：随机 IKM
	ikm := make([]byte, 32)
	_, _ = rand.Read(ikm)
	sk := blst.KeyGen(ikm) // 绑定提供 KeyGen
	// 序列化公钥（根据绑定，P1/P2 选择取决于签名约定）
	// 这里我们假设签名在 G2（常见于 IETF BLS_SIG_BLS12381G2），公钥在 G1
	pkObj := sk.P1() // 若 API 名称不同，请根据绑定 README 调整
	pkBytes := pkObj.Serialize()
	return &BlstBLS{
		sk: sk,
		pk: pkBytes,
		id: id,
	}
}

func (b *BlstBLS) Sign(message []byte) ([]byte, error) {
	// DST 选择必须与验证一致；示例使用常见 DST（IETF）
	const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
	sig := b.sk.Sign(message, []byte(dst), nil)
	return sig.Compress(), nil
}

func (b *BlstBLS) AggregateSignatures(sigs [][]byte) ([]byte, error) {
	if len(sigs) == 0 {
		return nil, nil
	}
	// 通过反序列化每个签名并聚合
	agg := blst.Aggregate{} // 注意：根据绑定版本可能是 NewAggregate() 或其它构造
	for _, sbytes := range sigs {
		var sig blst.Signature
		if err := sig.Deserialize(sbytes); err != nil {
			return nil, fmt.Errorf("sig deserialize: %w", err)
		}
		agg.Add(&sig)
	}
	aggSig := agg.ToSignature()
	return aggSig.Compress(), nil
}

func (b *BlstBLS) VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error) {
	if aggSig == nil {
		return false, fmt.Errorf("nil aggregate signature")
	}
	var sig blst.Signature
	if err := sig.Deserialize(aggSig); err != nil {
		return false, fmt.Errorf("agg deserialize: %w", err)
	}
	// 反序列化公钥列表为 P1Affine（示例假定公钥在 G1）
	pks := make([]*blst.P1Affine, 0, len(pubKeys))
	for _, pkb := range pubKeys {
		var p blst.P1Affine
		if err := p.Deserialize(pkb); err != nil {
			return false, fmt.Errorf("pk deserialize: %w", err)
		}
		pks = append(pks, &p)
	}
	const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"
	// 绑定通常提供聚合验证接口；下面示例调用 VerifyAggregate（API 名称视绑定而定）
	ok := sig.VerifyAggregate(pks, true, message, []byte(dst))
	return ok, nil
}

func (b *BlstBLS) PublicKey() []byte {
	return b.pk
}
