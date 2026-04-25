// Package wsclient 是 exchangeHubX WebSocket API 的轻量封装。
//
// 使用示例：
//
//	c := wsclient.New("wss://ws.j2coin.com/ws", appKey, secretKey)
//	if err := c.Dial(ctx); err != nil { ... }
//	defer c.Close()
//
//	c.OnMessage(func(raw []byte) { fmt.Println(string(raw)) })
//
//	if err := c.Auth(); err != nil { ... }         // 私有频道前先鉴权
//	if err := c.Subscribe("ticker@BTC_USDT"); err != nil { ... }
//
//	c.Run(ctx) // 阻塞：内部维护 ping 心跳 + 读循环，直到 ctx 取消或连接断开
package wsclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/exchangeHubX/open-api-demo-go/signer"
)

// PingInterval 与 PongTimeout 按文档建议值设置。
const (
	PingInterval = 20 * time.Second
	PongTimeout  = 30 * time.Second
)

// Handler 处理服务端推送的原始消息（已自动过滤掉 "pong" 文本帧）。
type Handler func(raw []byte)

// Client WebSocket 客户端
type Client struct {
	URL        string
	AppKey     string
	SecretKey  string
	RecvWindow int64 // 毫秒，默认 5000

	// Debug 打开后会把每一条发送出去的消息打到标准日志（包括 auth/订阅/心跳）。
	// 默认关闭。
	Debug bool

	conn    *websocket.Conn
	writeMu sync.Mutex // gorilla/websocket 要求写入串行化
	handler Handler
}

// New 创建客户端。url 形如 wss://ws.j2coin.com/ws。
func New(url, appKey, secretKey string) *Client {
	return &Client{
		URL:        url,
		AppKey:     appKey,
		SecretKey:  secretKey,
		RecvWindow: 5000,
	}
}

// OnMessage 注册消息处理器。未注册时消息会被丢弃。
func (c *Client) OnMessage(h Handler) { c.handler = h }

// Dial 建立 WebSocket 连接。
func (c *Client) Dial(ctx context.Context) error {
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, c.URL, nil)
	if err != nil {
		return fmt.Errorf("ws dial %s: %w", c.URL, err)
	}
	c.conn = conn
	return nil
}

// Close 关闭连接。
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// Auth 发送 op=auth 鉴权消息。签名由 signer.SignWSAuth 生成。
func (c *Client) Auth() error {
	return c.sendJSON(map[string]any{
		"op":   "auth",
		"args": []any{signer.SignWSAuth(c.AppKey, c.SecretKey, c.RecvWindow)},
	})
}

// Subscribe 订阅频道。例如 Subscribe("ticker@BTC_USDT", "depth@BTC_USDT,20")。
func (c *Client) Subscribe(channels ...string) error {
	return c.sendOp("subscribe", channels)
}

// Unsubscribe 取消订阅。
func (c *Client) Unsubscribe(channels ...string) error {
	return c.sendOp("unsubscribe", channels)
}

func (c *Client) sendOp(op string, args []string) error {
	return c.sendJSON(map[string]any{"op": op, "args": args})
}

func (c *Client) sendJSON(v any) error {
	if c.conn == nil {
		return errors.New("wsclient: not connected")
	}
	raw, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	return c.writeRaw(raw)
}

// sendPing 按文档要求以文本帧形式发送 "ping"。
func (c *Client) sendPing() error {
	return c.writeRaw([]byte("ping"))
}

func (c *Client) writeRaw(raw []byte) error {
	if c.Debug {
		log.Printf(">> %s", raw)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.conn.WriteMessage(websocket.TextMessage, raw)
}

// Run 启动心跳 + 读循环，阻塞直到 ctx 取消、连接异常或收到 close 帧。
func (c *Client) Run(ctx context.Context) error {
	if c.conn == nil {
		return errors.New("wsclient: not connected")
	}

	// 读超时：只要在 PongTimeout 内收到任意帧（包括 pong 文本）就算心跳健康
	resetReadDeadline := func() {
		_ = c.conn.SetReadDeadline(time.Now().Add(PongTimeout))
	}
	resetReadDeadline()

	errCh := make(chan error, 2)

	// ---- ping 协程 ----
	go func() {
		ticker := time.NewTicker(PingInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case <-ticker.C:
				if err := c.sendPing(); err != nil {
					errCh <- fmt.Errorf("send ping: %w", err)
					return
				}
			}
		}
	}()

	// ---- 读协程 ----
	go func() {
		for {
			_, msg, err := c.conn.ReadMessage()
			if err != nil {
				errCh <- fmt.Errorf("read: %w", err)
				return
			}
			resetReadDeadline()

			// 过滤 pong 文本帧
			if string(msg) == "pong" {
				continue
			}
			if c.handler != nil {
				c.handler(msg)
			}
		}
	}()

	// 任一协程出错即返回，并主动关闭连接以唤醒另一个
	err := <-errCh
	_ = c.conn.Close()
	return err
}
