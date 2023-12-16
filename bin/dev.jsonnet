{
  logger: {
    level: 'debug',
    // source: true,
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
      address: ':3433',
      certFile: 'test.crt',
      keyFile: 'test.key',
      close: '1s',
      dialer: 'unix+tls',
    },
  ],
}
