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
  * [portal-bridge](#portal-bridge)
  * [http-portal-bridge](#http-portal-bridge)
  * [udp-over-tcp](#udp-over-tcp)
* [udp](#udp)
* [sniproxy](#sniproxy)
* [logger](#logger)
* [pool](#pool)
* [api](#api)
* [fs](#fs)
* [docker](#docker)

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
      // 設置爲 true 則不驗證伺服器證書
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

從 v0.0.3 開始 websocket 支持 **fast** 屬性，如果設置爲 true，它將只使用 websocket 建立連接，在連接建立後直接使用 tcp 傳輸數據

從 v0.0.4 開始 http/websocket dialer 支持 **header**屬性( map\[string\]\[\]string ) 用於設置自定義的 http header

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

pipe 只能在同一進程中被使用，它直接在內存中模擬一個 net.Conn 因此效率非常高。它用於在進程內轉換流協議，例如 cloudflare 不支持顯示傳輸 http2協議，此時先將 http2 轉換爲 webscoekt 以經過 cloudflare 網路傳輸，在伺服器上將 websocekt 轉爲 http2 給伺服器 http2 服務。

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


# portal-bridge

有時我們需要將一個內網的服務映射到公網這時可以使用 portal-bridge 功能。

首先需要在有公網的伺服器上將一個 listener 的 mode 設置爲 'portal' 同時保證其 tag 唯一。之後可以在 dialer 中通過將 network 設置爲 'portal', 將 addr 設置爲 listener 的 tag 來創建連接。最後在內網伺服器上設置 bridge 數組反向連接 listener

```
// 這裏爲了測試方便將 portal/bridge 設置到了一起。通常真實環境 'portal' 位於公網伺服器，'bridge' 位於內網伺服器
local bridge = {
  dialer: [
    // 連接要發佈的服務
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
  ],
  // 'bridge' 會連接 'portal' 網路
  bridge: [
    {
      timeout: '200ms',
      url: 'basic://',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      // 提供 'bridge' 連接這個 dialer 到 'portal'
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
    },
  ],
};
// 這裏爲了測試方便將 portal/bridge 設置到了一起。通常真實環境 'portal' 位於公網伺服器，'bridge' 位於內網伺服器
local portal = {
  dialer: [
    // 這個 dialer 獲取到 tag 爲 'listener portal' 提供的連接
    {
      tag: 'portal',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      // connect portal tag
      addr: 'listener portal',
    },
  ],
  listener: [
    // 設置 mode 爲 'portal' 啓用 portal 網路
    {
      // 在 'portal' 模式下的監聽器必須保證 tag 唯一
      tag: 'listener portal',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      mode: 'portal',
      portal: {
        // 連接超時時間
        // Default 500ms
        timeout: '200ms',
        // 空閒連接每個多久發送一次心跳
        heart: '40s',
        // 等待心跳響應的超時時間
        heartTimeout: '1s',
      },
    },
    // 這個 listener 使用 'listener portal' 的連接提供服務
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'portal',
        close: '1s',
      },
    },
  ],
};
{
  dialer: bridge.dialer + portal.dialer,
  listener: portal.listener,
  bridge: bridge.bridge,
}
```

# http-portal-bridge

portal/bridge 也可以支持 http，並且 portal 模式的 listener 可以在 router 中可以混用 portal 和 普通的流量轉發

```
local bridge = {
  dialer: [
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
  ],
  bridge: [
    // websocket connect portal
    {
      timeout: '200ms',
      url: 'ws://example.com/http/ws',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      access: 'test access token',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
    },
    // http2 post connect portal
    {
      timeout: '200ms',
      url: 'http://example.com/http2',
      method: 'POST',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      access: 'test access token',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
    },
  ],
};
local portal = {
  dialer: [
    // serve by portal ws
    {
      tag: 'portal-ws',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      addr: 'listener-portal-ws',
    },
    // serve by portal http2
    {
      tag: 'portal-http2',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      addr: 'listener-portal-http2',
    },
    // serve direct
    {
      tag: 'portal-direct',
      timeout: '200ms',
      url: 'http://example.com/http/direct',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
    },
  ],
  listener: [
    {
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      mode: 'http',
      router: [
        // websocket portal
        {
          method: 'WS',
          pattern: '/http/ws',
          access: 'test access token',
          portal: {
            tag: 'listener-portal-ws',
            timeout: '200ms',
            heart: '40s',
            heartTimeout: '1s',
          },
        },
        // http2 portal
        {
          method: 'POST',
          pattern: '/http2',
          access: 'test access token',
          portal: {
            tag: 'listener-portal-http2',
            timeout: '200ms',
            heart: '40s',
            heartTimeout: '1s',
          },
        },
        // direct router
        {
          pattern: '/http/direct',
          dialer: {
            tag: 'tcp',
            close: '1s',
          },
        },
      ],
    },
    // portal-ws ingress
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'portal-ws',
        close: '1s',
      },
    },
    //  portal-http2 ingress
    {
      network: 'tcp',
      addr: ':4001',
      dialer: {
        tag: 'portal-http2',
        close: '1s',
      },
    },
    //  portal-direct ingress
    {
      network: 'tcp',
      addr: ':4002',
      dialer: {
        tag: 'portal-direct',
        close: '1s',
      },
    },
  ],
};
{
  dialer: bridge.dialer + portal.dialer,
  listener: portal.listener,
  bridge: bridge.bridge,
}
```

```
curl http://127.0.0.1:4000 http://127.0.0.1:4001 http://127.0.0.1:4002
```

# udp-over-tcp

從 v0.0.4 開始 basic 的 listener/dialer 支持 udp，這個功能用於實現 udp over tcp。這在某些時候是有用的，例如防火牆阻擋了 udp，或者 udp 被限制了速度，這時可以使用此功能來用 tcp 傳輸 udp 數據，但是注意這會降低原本 udp 程序的傳輸效率

```
// 這裏演示了如何實現 udp over tcp

// 這是伺服器上的配置，從tcp解析出udp，傳輸到目的服務
local server = {
  dialer: [
    {
      tag: 'google-dns',
      timeout: '200ms',
      url: 'basic://8.8.8.8:53',
      network: 'udp',
      udp: {
        frame: 16,
        timeout: '60s',
        size: 1500,
      },
    },
  ],
  listener: [
    {
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      dialer: {
        tag: 'google-dns',
        close: '1s',
      },
    },
  ],
};

// 這是一個反向代理。它接收udp，將其打包，並使用tcp傳輸到伺服器。
local proxy = {
  dialer: [
    {
      tag: 'udp-over-tcp',
      timeout: '200ms',
      url: 'basic://',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
    },
  ],
  listener: [
    {
      network: 'udp',
      addr: ':4000',
      dialer: {
        tag: 'udp-over-tcp',
        close: '1s',
      },
      udp: {
        frame: 16,
        timeout: '60s',
        size: 1500,
      },
    },
  ],
};

{
  dialer: server.dialer + proxy.dialer,
  listener: server.listener + proxy.listener,
  logger: {
    source: true,
  },
}
```

# udp
從 v0.0.5 開始支持 udp 數組用於指定一組 udp 端口映射

```
{
    udp:[
        {
            // udp listen host:port
            listen:":1053",
            // remote target addr
            to:"8.8.8.8:53",
            // udp max frame length, default 1024*2
            size:1500,
            // udp timeout, default 3m
            timeout:"3m",
        },
    ],
}
```

# sniproxy
從 v0.0.9 開始支持 sniproxy，它不會參與到 tls 加解密中去，它從客戶端讀取出 ClientHello 中的 sni，然後依據 sni 將流量原樣轉發到不同的後端。這可以爲不同 tls 後端提供一個共用的連接入口

sniproxy 還提供了一個 fallback 用於將非 tls 或未知的 tls 協議傳輸到一個回退的後端服務

```
{
  sniproxy: [
    // This listener receives tcp connections
    {
      network: 'tcp',
      addr: ':443',
      // Sniff sni timeout, Default 500ms
      timeout: '500ms',
      // Optionally dialer. for no matching SNI
      default: {
        tag: 'default',
        close: '1s',
      },
      // Optionally dialer. for non-TLS/unknown-TLS
      fallback: {
        tag: 'fallback',
        close: '1s',
      },
      // sni matching routes
      router: [
        {
          matcher: [
            {
              // - 'accuracy' is matched first and must be unique. This is the default value and you don't need to explicitly define type
              // - 'prefix' or 'suffix' will match the first matched route in the order configured after 'accuracy'
              // - 'regexp' will be matched last
              type: 'accuracy',
              value: 'www.bing.com',
            },
          ],
          dialer: {
            tag: 'bing',
            close: '1s',
          },
        },
        {
          matcher: [
            {
              value: 'www.google.com',
            },
          ],
          dialer: {
            tag: 'google',
            close: '1s',
          },
        },
      ],
    },
  ],
}
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

# api

你可以在 http 的 listener 中註冊 'API' 路由，她提供了一些 http 頁面用於查詢服務器運行狀態

```
{
  listener: [
    {
      network: 'tcp',
      addr: ':4000',
      mode: 'http',
      router: [
        {
          method: 'API',
          pattern: '/api',
          auth: [
            {
              username: 'dev',
              password: '123',
            },
          ],
        },
      ],
    },
  ],
}
```

# fs

fs 用於將一個操作系統目錄以靜態 http 的形式發佈到 http listener 的路由中，這不是這個程式的本職工作但這個需求很常見並且用 golang 實現毫不費力，所以也一起集成了此功能

```
local auth = [
  {
    username: 'dev',
    password: '123',
  },
];
{
  listener: [
    {
      network: 'tcp',
      addr: ':4000',
      mode: 'http',
      router: [
        {
          method: 'FS',
          pattern: '/fs',
          fs: '/tmp',
          auth: auth,
        },
        {
          method: 'API',
          pattern: '/',
          auth: auth,
        },
      ],
    },
  ],
}
```

# docker

```
docker run \
  -v Your_Configure_Path:/data/streamf.jsonnet:ro \
  -d king011/streamf:v0.0.1
```