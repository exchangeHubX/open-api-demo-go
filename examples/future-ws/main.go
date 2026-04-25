// examples/ws 演示如何使用 wsclient 连接 exchangeHubX WebSocket：
//
//	go run ./examples/ws
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/exchangeHubX/open-api-demo-go/wsclient"
)

func main() {
	const (
		// wsURL     = "wss://fws.j2coin.com/ws"
		wsURL     = "ws://127.0.0.1:10001/futures-ws"
		appKey    = "ak_5b595c06c56c16bebc60a7d9aa446b11"
		secretKey = "sk_7789e67dc58864d7a7cdc8ddc47264b1c507c141b76aa84361709698304effbf"
	)

	// Ctrl+C 优雅退出
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	c := wsclient.New(wsURL, appKey, secretKey)
	c.Debug = true // 打印所有发送出去的消息（包含 auth）
	c.OnMessage(func(raw []byte) {
		fmt.Printf("<< %s\n", raw)
	})

	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dialCancel()
	if err := c.Dial(dialCtx); err != nil {
		log.Fatalf("dial: %v", err)
	}
	defer c.Close()

	// 私有频道前先鉴权（公有频道如 ticker 可跳过）
	if err := c.Auth(); err != nil {
		log.Fatalf("auth: %v", err)
	}

	// 订阅公有行情 + 订单私有频道
	if err := c.Subscribe(
		"ticker@BTC_USDT",
		"depth@BTC_USDT,20",
		"kline@BTC_USDT,1m",
	); err != nil {
		log.Fatalf("subscribe: %v", err)
	}

	log.Println("ws running, press Ctrl+C to exit")
	if err := c.Run(ctx); err != nil && ctx.Err() == nil {
		log.Printf("ws exited: %v", err)
	}
}
