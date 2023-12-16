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
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'tcp://example.com',
      addr: 'localhost:2000',
    },
    {
      tag: 'tcp+tls',
      timeout: '200ms',
      url: 'tcp+tls://example.com',
      addr: 'localhost:2443',
      allowInsecure: true,
    },
    {
      tag: 'unix',
      timeout: '200ms',
      url: 'unix://@tcp-http.socket',
    },
    {
      tag: 'unix+tls',
      timeout: '200ms',
      url: 'unix+tls://@tcp-https.socket',
      allowInsecure: true,
    },
    {
      tag: 'ws',
      timeout: '200ms',
      url: 'ws://example.com/test/ws',
      addr: '@tcp-http.socket',
      network: 'unix',
    },
    {
      tag: 'wss',
      timeout: '200ms',
      url: 'wss://example.com/test/wss',
      addr: '@tcp-https.socket',
      network: 'unix',
      allowInsecure: true,
    },
  ],
  listener: [
    {
      network: 'unix',
      address: '@tcp-http.socket',
      close: '1s',
      dialer: 'tcp',
    },
    {
      network: 'unix',
      address: '@tcp-https.socket',
      certFile: 'test.crt',
      keyFile: 'test.key',
      close: '1s',
      dialer: 'tcp+tls',
    },
    {
      network: 'tcp',
      address: ':3000',
      close: '1s',
      dialer: 'unix',
    },
    {
      network: 'tcp',
      address: ':3443',
      certFile: 'test.crt',
      keyFile: 'test.key',
      close: '1s',
      dialer: 'unix+tls',
    },
  ],
}
