[English](README.md)
# streamf

這是一個用 golang 寫的端口轉發程序，但她不僅僅支持轉發端口數據，她還支持轉發數據流。即它可以將一種流協議數據轉發爲另外一種流協議

例如你可以將 tcp 數據流轉爲 websocket 數據流，反之亦然。

index:

* [run](#run)
  * [basic](#basic)
  * [http](#http)

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
      address: ':4000',
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
      address: ':4443',
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
curl -X PATCH http://127.0.0.1:4000/http2 -d 'abc=123'
```