// Use websocket in portal/bridge (example cloudflare network).
// Realize socket reuse by transmitting h2c in websocket.
local bridge = {
  dialer: [
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
    {
      tag: 'dialer-h2c',
      timeout: '200ms',
      url: 'basic://',
      network: 'pipe',
      addr: 'pipe.h2c',
    },
  ],
  bridge: [
    // connet portal by websocket
    {
      timeout: '400ms',
      url: 'ws://video.com/video/ws',
      network: 'tcp',
      addr: '127.0.0.1:4001',
      access: 'abc',
      dialer: {
        tag: 'dialer-h2c',
        close: '1s',
      },
    },
  ],
  listener: [
    // h2c is used to reuse sockets and routing
    {
      network: 'pipe',
      addr: 'pipe.h2c',
      mode: 'http',
      router: [
        {
          method: 'POST',
          pattern: '/video/secure',
          dialer: {
            tag: 'tcp',
            close: '1s',
          },
        },
      ],
    },
  ],
};
local portal = {
  dialer: [
    {
      tag: 'dialer',
      timeout: '1s',
      url: 'http://abc.com/video/secure',
      network: 'portal',
      addr: 'portal',
      method: 'POST',
      retry: 2,
    },
  ],
  listener: [
    {
      network: 'tcp',
      addr: ':4001',
      mode: 'http',
      router: [
        {
          method: 'API',
          pattern: '/',
          auth: [
            {
              username: 'dev',
              password: '123',
            },
          ],
        },
        {
          method: 'WS',
          pattern: '/video/ws',
          portal: {
            tag: 'portal',
            timeout: '1s',
            heart: '40s',
            heartTimeout: '1s',
          },
        },
      ],
    },
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'dialer',
        clsoe: '1s',
      },
    },
  ],
};
{
  dialer: bridge.dialer + portal.dialer,
  listener: bridge.listener + portal.listener,
  bridge: bridge.bridge,
}
