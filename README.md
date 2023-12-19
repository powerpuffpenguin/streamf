[中文](README.zh.md)
# streamf

This is a port forwarding program written in golang. But it doesn't just forward port data, it also supports forwarding streams. That is, data of a certain stream protocol is forwarded to another stream.

For example you can convert tcp data to websocket stream and vice versa.

index:

* [run](#run)
  * [basic](#basic)
  * [http](#http)
  * [unix](#unix)
  * [pipe](#pipe)
* [logger](#logger)
* [pool](#pool)

# run

Please use -conf to pass in the configuration file path to run the program

```
./streamf -conf your_configure_path
```

# basic
basic is the most basic forwarder. It is the tcp port forwarding program that can be found everywhere on the Internet.

```
// This is an example of port forwarding
{
  // Set a basic forwarding target
  dialer: [
    {
      // Use this tag in listener to specify dialer
      tag: 'tcp',
      // Connection timeout
      timeout: '200ms',
      // Connection URL, optional parameters network and addr override the addr in the URL
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'tcp+tls',
      timeout: '200ms',
      // +tls specifies to use tls to connect
      url: 'basic+tls://example.com',
      // Explicitly specify the connection addr
      network: 'tcp',
      addr: 'localhost:2443',
      // Do not verify server certificate
      allowInsecure: true,
    },
  ],
  listener: [
    // This listener receives tcp connections
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        // Forward to the dialer with tag 'tcp+tls'
        tag: 'tcp+tls',
        // After one end of the connection is disconnected, wait for one second before closing the other end
        //  (waiting for untransmitted data to continue transmitting)
        close: '1s',
      },
    },
    // This listener receives tls connections
    {
      network: 'tcp',
      addr: ':4443',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
      // Enable tls
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

http mode can support http in and out streams:

* websocket is supported in http1.1, which supports bidirectional data flow
* http2.0 also supports streaming for ordinary requests.

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
      // Receive WebSocket connection
      method: 'WS',
      // URL match pattern
      pattern: '/http/ws',
      dialer: {
        tag: 'wss',
        close: '1s',
      },
      // access specifies an access token, and only traffic matching the token is forwarded
      access: 'test access token',
    },
    {
      // Receive POST request
      method: 'POST',
      pattern: '/http2',
      dialer: {
        tag: 'ws',
        close: '1s',
      },
    },
    {
      // Receive PUT request
      method: 'PUT',
      pattern: '/http2',
      dialer: {
        tag: 'h2-post',
        close: '1s',
      },
      access: 'test access token',
    },
    {
      // Receive PATCH request
      method: 'PATCH',
      pattern: '/http2',
      dialer: { tag: 'h2c-put' },
    },
  ],
  listener: [
    {
      network: 'tcp',
      addr: ':4000',
      // Specify to use 'http' mode
      mode: 'http',
      // Specify route
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

> Incoming and outgoing traffic can be http1.x, but http1.x does not support data streaming and may wait until the end of the request or response traffic transmission before transmitting to the peer. http1.x is generally not recommended.

# unix

By default both incoming and outgoing traffic uses tcp, but you can set network to 'unix', which enables unix sockets, which are more efficient than sockets through the network card, but are only supported under linux

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

pipe can only be used in the same process. It simulates a net.Conn directly in memory and is therefore very efficient. It is used to convert the streaming protocol within the process. For example, cloudflare does not support the display transmission http2 protocol. At this time, http2 is first converted to webscoekt for transmission through the cloudflare network, and websocekt is converted to http2 on the server and then processed by the server http2 service.

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

logger is used to set logs

```
{
  logger: {
    // log level 'debug' 'info' 'warn' 'error'
    level: 'info',
    // Whether to display code files
    source: false,
  },
}
```

# pool

pool sets the read and write cache for the connection

```
{
  pool: {
    // Read and write cache size
    size: 1024 * 32,
    // How many free memory blocks can be cached at most?
    cache: 128,
  },
}
```
