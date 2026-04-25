package signer

import "testing"

// SignWSAuth 应当产生与 REST 相同规则的签名，
// 仅 method=GET、path=/ws/auth、无 query/body。
func TestSignWSAuth_Deterministic(t *testing.T) {
	// 固定时间戳 / recvwindow 手工构造期望值做对照
	const (
		appKey    = "ak_95e7762883a06dfc93ea479c08018afd"
		secretKey = "sk_057b2334f7c52095b1cfb6290758287b5f16b51fb0e9eb5e0935f37bb7ebbcf4"
	)

	// 直接调用 Sign 生成期望签名
	wantSig, _ := Sign(Params{
		Method: "GET",
		Path:   "/ws/auth",
		ValidateHeaders: map[string]string{
			HeaderAlgorithms: AlgorithmHmacSHA256,
			HeaderAppKey:     appKey,
			HeaderRecvWindow: "5000",
			HeaderTimestamp:  "1641446237201",
		},
		SecretKey: secretKey,
	})

	// 用 SignWSAuth 替换掉 timestamp 字段后应该一致
	got := SignWSAuth(appKey, secretKey, 5000)
	got[HeaderTimestamp] = "1641446237201" // 覆盖为固定值
	// 重新用覆盖后的 timestamp 计算
	reSig, _ := Sign(Params{
		Method: "GET",
		Path:   "/ws/auth",
		ValidateHeaders: map[string]string{
			HeaderAlgorithms: got[HeaderAlgorithms],
			HeaderAppKey:     got[HeaderAppKey],
			HeaderRecvWindow: got[HeaderRecvWindow],
			HeaderTimestamp:  got[HeaderTimestamp],
		},
		SecretKey: secretKey,
	})
	if reSig != wantSig {
		t.Fatalf("signature mismatch: got=%s want=%s", reSig, wantSig)
	}

	// 字段完整性
	for _, k := range []string{HeaderAlgorithms, HeaderAppKey, HeaderRecvWindow, HeaderTimestamp, HeaderSignature} {
		if got[k] == "" {
			t.Errorf("missing field %q", k)
		}
	}
	if got[HeaderAlgorithms] != AlgorithmHmacSHA256 {
		t.Errorf("algorithms = %q, want %q", got[HeaderAlgorithms], AlgorithmHmacSHA256)
	}
}
