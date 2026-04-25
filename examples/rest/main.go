package main

import (
	"fmt"
	"log"
	"net/url"

	"github.com/exchangeHubX/open-api-demo-go/client"
	"github.com/exchangeHubX/open-api-demo-go/signer"
)

func main() {
	// 示例 key，正式使用请换成自己的
	const (
		baseURL   = "http://127.0.0.1:10001"
		appKey    = "ak_5b595c06c56c16bebc60a7d9aa446b11"
		secretKey = "sk_7789e67dc58864d7a7cdc8ddc47264b1c507c141b76aa84361709698304effbf"
	)

	// ---- 1. 先用文档里的样例数据验证签名算法 ----
	verifyDocExample(appKey, secretKey)

	// ---- 2. 通过封装后的 Client 发起一次真实下单请求 ----
	c := client.New(baseURL, appKey, secretKey)
	c.RecvWindow = 60000

	status, body, err := c.Do(client.Request{
		Method: "POST",
		Path:   "/api/v1/orders",
		JSONBody: map[string]any{
			"symbol":      "JU_USDT",
			"side":        "BUY",
			"type":        "LIMIT",
			"timeInForce": "GTC",
			"bizType":     "SPOT",
			"price":       "0.1",
			"quantity":    "10",
		},
	})
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}
	fmt.Printf("\n[Real Request] status=%d\n%s\n", status, string(body))

	// ---- 3. GET + Query 示例 ----
	status, body, err = c.Do(client.Request{
		Method: "GET",
		Path:   "/api/v1/orders",
		Query: url.Values{
			"symbol": {"btc_usdt"},
		},
	})
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}
	fmt.Printf("\n[GET Query] status=%d\n%s\n", status, string(body))
}

// verifyDocExample 用文档中的样例值本地计算签名，用于自检。
// 文档样例：
//
//	original = validate-algorithms=HmacSHA256&validate-appkey=ak_...&validate-recvwindow=60000
//	           &validate-timestamp=1666026215729#POST#/api/v1/orders#{"symbol":"JU_USDT",...}
//	期望 signature = 017097d75f9506e2c6e6a074dd5a5556d4aefa8def40a455fe1240a9cd4e5ae9
func verifyDocExample(appKey, secretKey string) {
	body := `{"symbol":"BTC_USDT","side":"BUY","type":"LIMIT","timeInForce":"GTC","bizType":"SPOT","price":"0.1","quantity":"10"}`

	sig, original := signer.Sign(signer.Params{
		Method:   "POST",
		Path:     "/api/v1/orders",
		BodyJSON: body,
		ValidateHeaders: map[string]string{
			signer.HeaderAlgorithms: signer.AlgorithmHmacSHA256,
			signer.HeaderAppKey:     appKey,
			signer.HeaderRecvWindow: "60000",
			signer.HeaderTimestamp:  "1666026215729",
		},
		SecretKey: secretKey,
	})

	const expected = "017097d75f9506e2c6e6a074dd5a5556d4aefa8def40a455fe1240a9cd4e5ae9"
	fmt.Println("[Doc Example]")
	fmt.Println("original =", original)
	fmt.Println("signature=", sig)
	fmt.Println("expected =", expected)
	if sig == expected {
		fmt.Println("✅ signature matches doc example")
	} else {
		fmt.Println("❌ signature mismatch")
	}
}
