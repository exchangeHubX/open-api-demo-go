// Package client 对 Open API 的 HTTP 调用做简单封装：
// 自动填充 validate-* 头并生成签名。
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/exchangeHubX/open-api-demo-go/signer"
)

// Client Open API 客户端
type Client struct {
	BaseURL    string
	AppKey     string
	SecretKey  string
	RecvWindow int64 // 毫秒
	HTTPClient *http.Client
}

// New 创建客户端
func New(baseURL, appKey, secretKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		AppKey:     appKey,
		SecretKey:  secretKey,
		RecvWindow: 5000,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Request 一次 API 调用描述
type Request struct {
	Method string     // GET / POST / PUT / DELETE
	Path   string     // /api/v1/orders
	Query  url.Values // query 参数

	// 二选一：
	JSONBody any        // application/json，会 json.Marshal
	FormBody url.Values // application/x-www-form-urlencoded
}

// Do 发送请求并返回响应状态码和响应体
func (c *Client) Do(req Request) (int, []byte, error) {
	// 1. 准备 body
	var (
		bodyJSONStr string
		bodyFormVal url.Values
		bodyReader  io.Reader
		contentType string
	)
	switch {
	case req.JSONBody != nil:
		raw, err := json.Marshal(req.JSONBody)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal json body: %w", err)
		}
		bodyJSONStr = string(raw)
		bodyReader = bytes.NewReader(raw)
		contentType = "application/json"
	case len(req.FormBody) > 0:
		bodyFormVal = req.FormBody
		// 发送时 Go 标准库会做 URL 编码，和签名串保持一致
		encoded := req.FormBody.Encode()
		bodyReader = bytes.NewReader([]byte(encoded))
		contentType = "application/x-www-form-urlencoded"
	}

	// 2. 准备 validate-* 头
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	validateHeaders := map[string]string{
		signer.HeaderAlgorithms: signer.AlgorithmHmacSHA256,
		signer.HeaderAppKey:     c.AppKey,
		signer.HeaderRecvWindow: strconv.FormatInt(c.RecvWindow, 10),
		signer.HeaderTimestamp:  ts,
	}

	// 3. 生成签名
	sig, _ := signer.Sign(signer.Params{
		Method:          req.Method,
		Path:            req.Path,
		Query:           req.Query,
		BodyJSON:        bodyJSONStr,
		BodyForm:        bodyFormVal,
		ValidateHeaders: validateHeaders,
		SecretKey:       c.SecretKey,
	})

	// 4. 组装最终 URL
	fullURL := c.BaseURL + req.Path
	if len(req.Query) > 0 {
		fullURL += "?" + req.Query.Encode()
	}

	httpReq, err := http.NewRequest(req.Method, fullURL, bodyReader)
	if err != nil {
		return 0, nil, fmt.Errorf("new request: %w", err)
	}
	if contentType != "" {
		httpReq.Header.Set("Content-Type", contentType)
	}
	httpReq.Header.Set("Accept", "*/*")
	for k, v := range validateHeaders {
		httpReq.Header.Set(k, v)
	}
	httpReq.Header.Set(signer.HeaderSignature, sig)

	// 5. 发送
	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return 0, nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("read body: %w", err)
	}
	return resp.StatusCode, respBody, nil
}
