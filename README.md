[中文](README.zh.md)
# streamf

This is a port forwarding program written in golang. But it doesn't just forward port data, it also supports forwarding streams. That is, data of a certain stream protocol is forwarded to another stream.

For example you can convert tcp data to websocket stream and vice versa.

index:

* [run](#run)
  * [basic](#basic)
  * [http](#http)

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
      // Connection URL, optional parameters network and addr override the address in the URL
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'tcp+tls',
      timeout: '200ms',
      // +tls specifies to use tls to connect
      url: 'basic+tls://example.com',
      // Explicitly specify the connection address
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
      address: ':4000',
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
      address: ':4443',
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
curl -X PATCH http://127.0.0.1:4000/http2 -d 'abc=123'
```