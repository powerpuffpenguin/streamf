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
