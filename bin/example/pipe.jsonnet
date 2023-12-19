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
