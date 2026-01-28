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

// BlstBLS 抽象出使用 blst 绑定的最小 BLS 操作集合。
// - sk: 私钥对象（blst.SecretKey 指针，具体类型以绑定实现为准）
// - pk: 公钥的序列化字节切片（便于网络传输/持久化）
// - id: 节点或签名者的标识符（用于上层协议，非 BLS 本身需要）
type BlstBLS struct {
	sk *blst.SecretKey
	pk []byte
	id int
}

// NewBlstBLS 创建并初始化一个 BlstBLS 实例。
// 说明：示例中用随机 IKM（input key material）生成私钥，仅用于测试/演示。
// 生产应通过安全方式（KMS / 密钥文件 / 环境隔离）加载/生成私钥。
func NewBlstBLS(id int) *BlstBLS {
	// 生成随机 IKM（32 字节）——仅示例用途
	ikm := make([]byte, 32)
	_, _ = rand.Read(ikm)

	// 调用绑定的 KeyGen 从 IKM 产生 SecretKey。
	// 注意：不同版本的绑定可能有不同的函数签名或命名��例如 KeyGen/KeyGenWithInfo 等）。
	// 若编译失败，请查阅绑定仓库 README 并适配。
	sk := blst.KeyGen(ikm) // 绑定提供 KeyGen

	// 根据常见约定：签名在 G2，公钥在 G1（IETF BLS sig 使用此分配）。
	// 这里��用 sk.P1()（或相似方法）得到公钥对象，然后序列化为字节。
	// 如果绑定使用 P2 作为公钥（或 API 名称不同），需要相应调整。
	pkObj := sk.P1() // 若 API 名称不同，请根据绑定 README 调整
	pkBytes := pkObj.Serialize()

	return &BlstBLS{
		sk: sk,
		pk: pkBytes,
		id: id,
	}
}

// Sign 使用实例持有的私钥对消息进行签名并返回压缩后的签名字节。
// - message: 待签消息（字节切片），上层应保证消息构造一致（例如序列化顺序）
// - 返回值为压缩（Compressed）形式的签名字节，便于传输/存储
func (b *BlstBLS) Sign(message []byte) ([]byte, error) {
	// DST（domain separation tag）必须与验证时一致。
	// 这里选用 IETF 推荐的一种常见 DST（用于 BLS_SIG_BLS12381G2）
	// 注意：生产系统务必确认所用的签名/映射到曲线的参数与 DST。
	const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"

	// 调用绑定的 Sign：将消息、DST（和可选的扩展）传入。
	// 返回的 sig 类型通常是 *blst.Signature 或 blst.Signature。
	sig := b.sk.Sign(message, []byte(dst), nil)

	// 使用 Compress/Serialize/Compress 方法将签名变为字节便于网络传输。
	// 这里使用 Compress()（绑定 API 可能命名为 Compress/SerializeCompressed 等）。
	return sig.Compress(), nil
}

// AggregateSignatures 将多个单独签名聚合成一个聚合签名并返回其序列化（压缩）形式。
// - sigs: 每项为单个签名的字节表示（已经是序列化/压缩形式）
// - 返回聚合签名的字节；若输入为空，返回 nil（表示没有签名）
func (b *BlstBLS) AggregateSignatures(sigs [][]byte) ([]byte, error) {
	// 若没有任何签名，直接返回 nil（上层可据此判断聚合失败或未达阈值）
	if len(sigs) == 0 {
		return nil, nil
	}

	// 下面演示如何反序列化每个签名并将其加入聚合器。
	// 不同绑定版本的 API 可能提供 Aggregate 类型或直接的静态聚合函数（如 blst.Aggregate/Combine）。
	agg := blst.Aggregate{} // 注意：根据绑定版本可能需要改为 blst.NewAggregate() 等

	for _, sbytes := range sigs {
		// 反序列化签名字节到签名对象
		var sig blst.Signature
		if err := sig.Deserialize(sbytes); err != nil {
			// 若某个签名无法反序列化，返回错误以便上层处理（比如记录坏节点）
			return nil, fmt.Errorf("sig deserialize: %w", err)
		}
		// 将反序列化后的签名添加到聚合器
		agg.Add(&sig)
	}

	// 将聚合器内容导出为单个签名对象（绑定 API 可能为 ToSignature/ToSig/Finalize 等）
	aggSig := agg.ToSignature()
	// 返回压缩格式的聚合签名（字节）
	return aggSig.Compress(), nil
}

// VerifyAggregate 验证一个对于同一消息由多个公钥签名而生成的聚合签名。
// - pubKeys: 公钥字节数组的切片（每项为序列化形式，假定在 G1）
// - message: 被签名的消息（字节）
// - aggSig: 聚合签名的字节表示（序列化/压缩形式）
// 返回：bool 表示验证是否通过，error 表示过程中的解析或调用错误
func (b *BlstBLS) VerifyAggregate(pubKeys [][]byte, message []byte, aggSig []byte) (bool, error) {
	// 空聚合签名直接视为错误（可以按需调整为 false, nil）
	if aggSig == nil {
		return false, fmt.Errorf("nil aggregate signature")
	}

	// 反序列化聚合签名字节到签名对象
	var sig blst.Signature
	if err := sig.Deserialize(aggSig); err != nil {
		// 解析失败通常意味着签名数据损坏或格式不匹配
		return false, fmt.Errorf("agg deserialize: %w", err)
	}

	// 将每个公钥字节反序列化为 P1Affine（示例中假定公钥在 G1）
	// 如果公钥在 G2，则需使用 P2Affine 等类型并相应调整验证调用。
	pks := make([]*blst.P1Affine, 0, len(pubKeys))
	for _, pkb := range pubKeys {
		var p blst.P1Affine
		if err := p.Deserialize(pkb); err != nil {
			// 反序列化失败表明公钥数据有问题
			return false, fmt.Errorf("pk deserialize: %w", err)
		}
		// 将反序列化后的 P1Affine 指针追加到切片
		pks = append(pks, &p)
	}

	// 使用与签名时相同的 DST（域分离标签）
	const dst = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_NUL_"

	// 调用绑定提供的聚合验证函数。
	// 这里示例使用 sig.VerifyAggregate(pks, true, message, []byte(dst))
	// - 第一个参数是公钥列表（P1Affine 指针切片）
	// - 第二个布尔值可能表示是否对消息进行预哈希（具体含义依绑定而定）
	// - 剩余参数为消息与 DST
	// 不同绑定版本的函数签名可能不同，请参阅绑定 README 以确保正确调用。
	ok := sig.VerifyAggregate(pks, true, message, []byte(dst))
	return ok, nil
}

// PublicKey 返回实例的公钥字节（序列化形式）
// 上层协议可将此字节广播或保存以便验证签名时使用
func (b *BlstBLS) PublicKey() []byte {
	return b.pk
}
