package signer

import (
	"net/url"
	"testing"
)

// 文档 sign-code.md 中的样例
func TestSign_DocExample(t *testing.T) {
	body := `{"symbol":"JU_USDT","side":"BUY","type":"LIMIT","timeInForce":"GTC","bizType":"SPOT","price":"0.1","quantity":"10"}`

	sig, original := Sign(Params{
		Method:   "POST",
		Path:     "/api/v1/orders",
		BodyJSON: body,
		ValidateHeaders: map[string]string{
			HeaderAlgorithms: AlgorithmHmacSHA256,
			HeaderAppKey:     "ak_95e7762883a06dfc93ea479c08018afd",
			HeaderRecvWindow: "60000",
			HeaderTimestamp:  "1666026215729",
		},
		SecretKey: "sk_057b2334f7c52095b1cfb6290758287b5f16b51fb0e9eb5e0935f37bb7ebbcf4",
	})

	wantOriginal := `validate-algorithms=HmacSHA256&validate-appkey=ak_95e7762883a06dfc93ea479c08018afd&validate-recvwindow=60000&validate-timestamp=1666026215729#POST#/api/v1/orders#` + body
	if original != wantOriginal {
		t.Errorf("original mismatch:\n got: %s\nwant: %s", original, wantOriginal)
	}

	const wantSig = "017097d75f9506e2c6e6a074dd5a5556d4aefa8def40a455fe1240a9cd4e5ae9"
	if sig != wantSig {
		t.Errorf("signature mismatch:\n got: %s\nwant: %s", sig, wantSig)
	}
}

func TestSign_QueryOnly(t *testing.T) {
	_, original := Sign(Params{
		Method: "GET",
		Path:   "/api/v1/orders",
		Query: url.Values{
			"symbol": {"btc_usdt"},
			"side":   {"BUY"},
		},
		ValidateHeaders: map[string]string{
			HeaderAlgorithms: AlgorithmHmacSHA256,
			HeaderAppKey:     "ak",
			HeaderRecvWindow: "5000",
			HeaderTimestamp:  "1",
		},
		SecretKey: "sk",
	})

	want := "validate-algorithms=HmacSHA256&validate-appkey=ak&validate-recvwindow=5000&validate-timestamp=1#GET#/api/v1/orders#side=BUY&symbol=btc_usdt"
	if original != want {
		t.Errorf("original mismatch:\n got: %s\nwant: %s", original, want)
	}
}
