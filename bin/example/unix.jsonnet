// curl https://127.0.0.1:4443/test/tls http://127.0.0.1:4000/test/tcp  -k
// This is an example of port forwarding
{
  logger: {
    level: 'debug',
    // source: true,
  },
  pool: {
    size: 1024 * 32,
    cache: 128,
  },
  dialer: [
    // Set a basic forwarding target
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'tcp+tls',
      timeout: '200ms',
      url: 'basic+tls://example.com?addr=localhost:2443',
      allowInsecure: true,
    },
  ],
  listener: [
    // This listener receives tcp connections
    {
      network: 'tcp',
      address: ':4000',
      dialer: {
        // Forward to the dialer with tag 'tcp'
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
