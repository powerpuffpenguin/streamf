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
  * [portal-bridge](#portal-bridge)
  * [http-portal-bridge](#http-portal-bridge)
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

# portal-bridge

Sometimes we need to publish an intranet service to the public network. In this case, we can use the portal-bridge function.

First, you need to set the mode of a listener to 'portal' on a server with a public network and ensure that its tag is unique. You can then create a connection in dialer by setting network to 'portal' and addr to listener's tag. Finally, set the bridge array reverse connection listener on the intranet server.

```
// The portal/bridge are set together for the convenience of testing. Usually in the real environment, 'portal' is located on the public network server, and 'bridge' is located on the intranet server.
local bridge = {
  dialer: [
    // Connect to the service you want to publish
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
  ],
  // The 'bridge' will connect to the 'portal' network
  bridge: [
    {
      timeout: '200ms',
      url: 'basic://',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      // Connect this dialer to the 'portal' through the 'bridge'
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
    },
  ],
};
// The portal/bridge are set together for the convenience of testing. Usually in the real environment, 'portal' is located on the public network server, and 'bridge' is located on the intranet server.
local portal = {
  dialer: [
    // This dialer will obtain the connection provided by the 'listener portal'
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
    // Set mode to 'portal' to enable portal networking
    {
      // Listeners in 'portal' mode must have a unique tag
      tag: 'listener portal',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      mode: 'portal',
      portal: {
        // Wait connect timeout
        // Default 500ms
        timeout: '200ms',
        // How often does an idle connection send a heartbeat?
        heart: '40s',
        // Timeout for waiting for heartbeat response
        heartTimeout: '1s',
      },
    },
    // This listener uses the connection provided by the 'listener portal' to provide services to the outside world.
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

portal/bridge can also support http, and the portal mode listener can be used in the router to mix portal and ordinary traffic forwarding.

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