// The portal/bridge are set together for the convenience of testing. Usually in the real environment, 'portal' is located on the public network server, and 'bridge' is located on the intranet server.
local bridge = {
  dialer: [
    // Connect to the service you want to publish
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
  ],
  // The 'bridge' will connect to the 'portal' network
  bridge: [
    {
      timeout: '200ms',
      url: 'ws://example.com/http/ws',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      access: 'test access token',
      // Connect this dialer to the 'portal' through the 'bridge'
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
    },
  ],
};
// The portal/bridge are set together for the convenience of testing. Usually in the real environment, 'portal' is located on the public network server, and 'bridge' is located on the intranet server.
local portal = {
  dialer: [
    // This dialer will obtain the connection provided by the 'listener portal'
    {
      tag: 'portal-ws',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      // connect portal tag
      addr: 'listener-portal-ws',
    },
    {
      tag: 'portal-direct',
      timeout: '200ms',
      url: 'http://example.com/http/direct',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      method: 'POST',
    },
    {
      tag: 'wss',
      timeout: '200ms',
      url: 'wss://example.com',
      addr: 'localhost:2443',
      allowInsecure: true,
    },
  ],
  listener: [
    {
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      mode: 'http',
      router: [
        {
          method: 'WS',
          pattern: '/http/ws',
          access: 'test access token',
          // enable portal networking
          portal: {
            tag: 'listener-portal-ws',
            // Wait connect timeout
            // Default 500ms
            timeout: '200ms',
            // How often does an idle connection send a heartbeat?
            heart: '40s',
            // Timeout for waiting for heartbeat response
            heartTimeout: '1s',
          },
        },
        {
          method: 'POST',
          pattern: '/http/direct',
          dialer: { tag: 'wss' },
        },
      ],
    },
    // This listener uses the connection provided by the 'listener portal' to provide services to the outside world.
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'portal-ws',
        close: '1s',
      },
    },
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
  // logger: {
  //   log: 'debug',
  //   source: true,
  // },
  dialer: bridge.dialer + portal.dialer,
  listener: portal.listener,
  bridge: bridge.bridge,
}
