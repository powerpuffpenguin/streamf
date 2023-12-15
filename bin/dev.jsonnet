{
  logger: {
    level: 'debug',
    // source: true,
  },
  dialer: [
    {
      tag: 'http',
      timeout: '200ms',
      url: 'tcp://example.com',
      addr: 'localhost:2000',
    },
    {
      tag: 'https',
      timeout: '200ms',
      url: 'tcp+tls://example.com',
      addr: 'localhost:2443',
      allowInsecure: true,
    },
  ],
  listener: [
    {
      network: 'unix',
      address: '@http.socket',
      close: '1s',
      dialer: 'http',
    },
  ],
}
