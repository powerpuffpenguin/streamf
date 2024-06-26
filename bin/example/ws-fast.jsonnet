{
  dialer: [
    {
      tag: 'tls',
      timeout: '200ms',
      url: 'basic+tls://example.com',
      network: 'tcp',
      addr: 'localhost:2443',
      allowInsecure: true,
    },
    {
      tag: 'pipe.ws',
      timeout: '200ms',
      url: 'ws://example.com/ws',
      network: 'pipe',
      addr: 'http.pipe',
      fast: true,
    },
    {
      tag: 'portal',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      addr: 'listener-portal-ws',
    },
  ],
  bridge: [
    {
      timeout: '200ms',
      url: 'ws://example.com/portal',
      fast: true,
      network: 'pipe',
      addr: 'http.pipe',
      dialer: {
        tag: 'tls',
        close: '1s',
      },
    },
  ],
  listener: [
    {
      network: 'tcp',
      addr: ':9000',
      dialer: {
        tag: 'pipe.ws',
        close: '1s',
      },
    },
    {
      network: 'tcp',
      addr: ':9001',
      dialer: {
        tag: 'portal',
        close: '1s',
      },
    },
    {
        network: 'pipe',
        addr: 'http.pipe',
        mode: 'http',
        router: [
            {
                method: 'WS',
                pattern: '/ws',
                fast :true,
                dialer: {
                    tag: 'tls',
                    close: '1s',
                },
            },
            {
                method: 'WS',
                pattern: '/portal',
                fast :true,
                portal: {
                    tag: 'listener-portal-ws',
                    timeout: '200ms',
                    heart: '40s',
                    heartTimeout: '1s',
                },
            },
        ],
    },
  ],
}
