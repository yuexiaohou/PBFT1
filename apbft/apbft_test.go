package pbft // 定义包为 main（可执行包）

// 导入需要的包
import (
	"math/rand" // 导入用于生成随机数的包
	"testing"   // 导入用于编写测试的包
	"time"      // 导入用于处理时间的包
)

// 定义一个测试函数，Go 的测试以 Test 开头并接受 *testing.T 参数
func TestPBFTSimulationWithStub(t *testing.T) {
	rand.Seed(time.Now().UnixNano()) // 使用当前时间的纳秒数作为随机数种子，确保每次运行随机性不同
	nodes := []*Node{}               // 初始化一个空的 Node 指针切片，用于存放模拟中的节点
	for i := 0; i < 7; i++ {          // 循环创建 7 个节点（i 从 0 到 6）
		tp := 80.0 + rand.Float64()*120.0 // 为每个节点生成一个随机的处理时间（或权重），范围大概在 80 到 200 之间
		isMal := false                   // 默认为非恶意节点
		if i == 3 {                      // 将索引为 3 的节点设为恶意节点（模拟容错场景中的坏节点）
			isMal = true
		}
		nodes = append(nodes, NewNode(i, tp, isMal, false)) // 创建节点并追加到 nodes 切片，最后一个参数为 false（可能表示是否为备用或其它标记）
	}
	sim := NewPBFTSimulator(nodes, false) // 使用创建的节点切片初始化 PBFT 模拟器，第二个参数为 false（可能为调试或启用外部 stub 的标志）
	sim.ComputeTiers()                    // 计算模拟器中节点的分层或分组信息（例如主备、委托关系等）

	rounds := 5          // 要运行的共识轮数设为 5
	successes := 0       // 记录成功达成共识的轮次数
	for r := 0; r < rounds; r++ {                       // 循环执行每一轮共识
		if sim.RunRound(r, []byte("test-request")) {    // 运行第 r 轮，发送的请求为字节切片 "test-request"
			successes++                                  // 如果该轮达成共识，则成功计数加一
		}
	}
	if successes == 0 {                                 // 如果所有轮次都未成功达成共识
		t.Fatalf("expected at least 1 successful consensus round, got 0") // 则标记测试失败并打印错误信息
	}
} // 测试函数结束
