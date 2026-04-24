// Package signer 实现 exchangeHubX Open API 的请求签名逻辑。
//
// 签名规则参考 docs/common/sign-code.md：
//
//  1. 数据部分 Y：
//     Y = #method#path[#query][#body]
//     - method: 大写的 HTTP 方法
//     - path:   请求路径（restful 参数填充后的真实路径）
//     - query:  按 key 字典序排序后的 k=v&k=v 拼接
//     - body:
//     application/json           -> 原始 JSON 字符串，不重新排序
//     application/x-www-form-... -> 按 key 字典序排序后的 k=v&k=v 拼接
//
//  2. 请求头部分 X：
//     将所有 validate-* 签名头（不包含 validate-signature 本身）
//     按 key 字典序升序，使用 & 拼接成 k=v&k=v。
//
//  3. 最终待签名串 original = X + Y
//     signature = HmacSHA256Hex(secretKey, original)
//
//  4. 将签名放到请求头 validate-signature 中。
package signer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
)

// Header 常量
const (
	HeaderAlgorithms = "validate-algorithms"
	HeaderAppKey     = "validate-appkey"
	HeaderRecvWindow = "validate-recvwindow"
	HeaderTimestamp  = "validate-timestamp"
	HeaderSignature  = "validate-signature"

	AlgorithmHmacSHA256 = "HmacSHA256"
)

// Params 签名所需的全部输入。
type Params struct {
	// Method HTTP 方法，会被强制转为大写
	Method string
	// Path 请求路径，例如 /api/v1/orders
	Path string
	// Query 查询参数；若 URL 中本身带有 query，也可以直接传一个已解析好的 url.Values
	Query url.Values
	// Body 请求体：
	//   - JSON      : 直接填原始 JSON 字符串
	//   - x-www-form: 传 url.Values，由签名器自动排序拼接。为避免和 JSON 混淆，
	//                 这里分成 BodyJSON / BodyForm 两个字段
	BodyJSON string
	BodyForm url.Values
	// ValidateHeaders 参与签名的 validate-* 头（不含 validate-signature）
	ValidateHeaders map[string]string
	// SecretKey 用户的 secretKey
	SecretKey string
}

// Sign 根据 Params 生成签名，返回 (signature, original 待签名串)。
// original 返回出来主要方便调试对比。
func Sign(p Params) (signature string, original string) {
	y := buildDataPart(p)
	x := buildHeaderPart(p.ValidateHeaders)
	original = x + y
	signature = hmacSHA256Hex(p.SecretKey, original)
	return
}

// buildDataPart 构造 Y = #method#path[#query][#body]
func buildDataPart(p Params) string {
	var b strings.Builder

	b.WriteByte('#')
	b.WriteString(strings.ToUpper(p.Method))

	b.WriteByte('#')
	b.WriteString(p.Path)

	if q := encodeSortedValues(p.Query); q != "" {
		b.WriteByte('#')
		b.WriteString(q)
	}

	switch {
	case p.BodyJSON != "":
		b.WriteByte('#')
		b.WriteString(p.BodyJSON)
	case len(p.BodyForm) > 0:
		b.WriteByte('#')
		b.WriteString(encodeSortedValues(p.BodyForm))
	}

	return b.String()
}

// buildHeaderPart 构造 X = 按 key 升序的 validate-* 头的 k=v&k=v 拼接
func buildHeaderPart(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		// validate-signature 不参与签名
		if strings.EqualFold(k, HeaderSignature) {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+headers[k])
	}
	return strings.Join(parts, "&")
}

// encodeSortedValues 按 key 字典序排序，拼接成 k=v&k=v。
// 注意：文档中的示例未对 value 做 URL 编码，这里也保持原样，避免签名端不一致。
// 如果服务端实际做了 URL 编码，请根据服务端实现调整。
func encodeSortedValues(v url.Values) string {
	if len(v) == 0 {
		return ""
	}
	keys := make([]string, 0, len(v))
	for k := range v {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		for _, val := range v[k] {
			parts = append(parts, k+"="+val)
		}
	}
	return strings.Join(parts, "&")
}

func hmacSHA256Hex(secret, data string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
