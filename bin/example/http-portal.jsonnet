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
