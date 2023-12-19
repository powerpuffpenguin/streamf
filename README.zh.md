[English](README.md)
# streamf

這是一個用 golang 寫的端口轉發程序，但她不僅僅支持轉發端口數據，她還支持轉發數據流。即它可以將一種流協議數據轉發爲另外一種流協議

例如你可以將 tcp 數據流轉爲 websocket 數據流，反之亦然。

index:

* [run](#run)
  * [basic](#basic)
  * [http](#http)
  * [unix](#unix)
  * [pipe](#pipe)
* [logger](#logger)
* [pool](#pool)

# run

請使用 -conf 傳入設定檔案路徑運行程式

```
./streamf -conf your_configure_path
```

# basic
basic 是最基礎的轉發器，她就是網路上隨處可見 tcp 端口轉發程式

```
// 這是一個端口轉發例子
{
  // 設置 basic 模式的轉發目標
  dialer: [
    {
      // 在 listener 中使用這個 tag 來指定這個 dialer
      tag: 'tcp',
      // 連接超時時間
      timeout: '200ms',
      // 要連接的 URL, 可以使用可選參數 network 和 addr 來覆蓋 URL 中的連接地址
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'tcp+tls',
      timeout: '200ms',
      // +tls 指定使用 tls 連接
      url: 'basic+tls://example.com',
      // 明確指定連接地址
      network: 'tcp',
      addr: 'localhost:2443',
      // 設置爲 true 則不驗證服務器證書
      allowInsecure: true,
    },
  ],
  listener: [
    // 這個 listener 接收 tcp 連接
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        // 將數據轉發給 tag 爲 'tcp+tls' 的 dialer
        tag: 'tcp+tls',
        // 在一端的連接端口後，等待多久再關閉另外一端
        // (用於等待未傳輸完的數據，傳輸完成)
        close: '1s',
      },
    },
    // 這個 listener 接收 tls 連接
    {
      network: 'tcp',
      addr: ':4443',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
      // 啓用 tls
      tls: {
        certFile: 'test.crt',
        keyFile: 'test.key',
      },
    },
  ],
}
```

```
curl https://127.0.0.1:4443/test/tls http://127.0.0.1:4000/test/tcp  -k
```

# http

http 模式能夠支持 http 入棧和出棧流:

* websocket 在 http1.1 中被支持，它支持雙向數據流
* http2.0 對普通請求也支持了流

```
{
  dialer: [
    {
      tag: 'wss',
      timeout: '200ms',
      url: 'wss://example.com',
      addr: 'localhost:2443',
      allowInsecure: true,
    },
    {
      tag: 'ws',
      timeout: '200ms',
      url: 'ws://example.com/http/ws',
      addr: 'localhost:4000',
      access: 'test access token',
    },
    {
      tag: 'h2-post',
      url: 'https://example.com/http2',
      addr: 'localhost:4443',
      allowInsecure: true,
    },
    {
      tag: 'h2c-put',
      url: 'http://example.com/http2',
      addr: 'localhost:4000',
      method: 'PUT',
      access: 'test access token',
    },
  ],
  local router = [
    {
      // 接受 WebSocket 連接
      method: 'WS',
      // URL match pattern
      pattern: '/http/ws',
      dialer: {
        tag: 'wss',
        close: '1s',
      },
      // access 用於指定訪問 token，如果設置則只有 token 匹配的流量才會被轉發
      access: 'test access token',
    },
    {
      // 接受 POST 請求
      method: 'POST',
      pattern: '/http2',
      dialer: {
        tag: 'ws',
        close: '1s',
      },
    },
    {
      // 接受 PUT 請求
      method: 'PUT',
      pattern: '/http2',
      dialer: {
        tag: 'h2-post',
        close: '1s',
      },
      access: 'test access token',
    },
    {
      // 接受 PATCH 請求
      method: 'PATCH',
      pattern: '/http2',
      dialer: { tag: 'h2c-put' },
    },
  ],
  listener: [
    {
      network: 'tcp',
      addr: ':4000',
      // 指定使用 http 模式
      mode: 'http',
      // 爲 http 指定路由
      router: router,
    },
    {
      network: 'tcp',
      addr: ':4443',
      mode: 'http',
      router: router,
      tls: {
        certFile: 'test.crt',
        keyFile: 'test.key',
      },
    },
  ],
}
```

```
curl -X PATCH http://127.0.0.1:4000/http2 -d 'abc=123'
```

> 流入和流出流量可以是 http1.x，但是 http1.x 並不支持數據流，它可能會等到請求或響應流量傳輸完畢才傳輸到對端。通常不建議使用 http1.x

# unix

默認流入和流出流量都使用 tcp，但是你可以設置 network 爲 unix，這樣可以啓用 unix socket，它比經過網卡的 socket 更高效，但是它只在 linux 下被支持

```
{
  dialer: [
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'unix+tls',
      timeout: '200ms',
      url: 'basic+tls://example.com',
      network: 'unix',
      addr: '@streamf/unix.socket',
      allowInsecure: true,
    },
  ],
  listener: [
    {
      network: 'unix',
      addr: '@streamf/unix.socket',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
      tls: {
        certFile: 'test.crt',
        keyFile: 'test.key',
      },
    },
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'unix+tls',
        close: '1s',
      },
    },
  ],
}
```

```
curl  http://127.0.0.1:4000/
curl --abstract-unix-socket streamf/unix.socket  https://example.com/ -k
```

# pipe

pipe 只能在同一進程中被使用，它直接在內存中模擬一個 net.Conn 因此效率非常高。它用於在進程內轉換流協議，例如 cloudflare 不支持顯示傳輸 http2協議，此時先將 http2 轉換爲 webscoekt 以經過 cloudflare 網路傳輸，在服務器上將 websocekt 轉爲 http2 給服務器 http2 服務。

```
{
  dialer: [
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'pipe+tls',
      timeout: '200ms',
      url: 'basic+tls://example.com',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      allowInsecure: true,
    },
  ],
  listener: [
    {
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
      tls: {
        certFile: 'test.crt',
        keyFile: 'test.key',
      },
    },
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'pipe+tls',
        close: '1s',
      },
    },
  ],
}
```

```
curl  http://127.0.0.1:4000/
```

# logger

logger 用於設定日誌

```
{
  logger: {
    // 日誌等級 'debug' 'info' 'warn' 'error'
    level: 'info',
    // 是否顯示代碼檔案
    source: false,
  },
}
```

# pool

pool 爲連接設置讀寫緩存

```
{
  pool: {
    // 讀寫緩存大小
    size: 1024 * 32,
    // 最多緩存多少個空閒的內存塊
    cache: 128,
  },
}
```
