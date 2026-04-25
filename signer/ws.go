package signer

import (
	"strconv"
	"time"
)

// SignWSAuth 构造 WebSocket `op=auth` 请求所需的 validate-* 字段集合。
//
// 根据 docs/common/websocket.md，WS 鉴权的签名规则与 REST 完全一致，
// 仅约定：method = GET，path = /ws/auth，无 query、无 body。
//
// 返回的 map 可直接作为 auth 请求的 args[0]。
func SignWSAuth(appKey, secretKey string, recvWindow int64) map[string]string {
	headers := map[string]string{
		HeaderAlgorithms: AlgorithmHmacSHA256,
		HeaderAppKey:     appKey,
		HeaderRecvWindow: strconv.FormatInt(recvWindow, 10),
		HeaderTimestamp:  strconv.FormatInt(time.Now().UnixMilli(), 10),
	}
	sig, _ := Sign(Params{
		Method:          "GET",
		Path:            "/ws/auth",
		ValidateHeaders: headers,
		SecretKey:       secretKey,
	})
	headers[HeaderSignature] = sig
	return headers
}
