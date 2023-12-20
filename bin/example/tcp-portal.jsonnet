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
      url: 'basic://',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
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
      tag: 'portal',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      // connect portal tag
      addr: 'listener portal',
    },
  ],
  listener: [
    // Set mode to 'portal' to enable portal networking
    {
      // Listeners in 'portal' mode must have a unique tag
      tag: 'listener portal',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      mode: 'portal',
      portal: {
        // Wait connect timeout
        // Default 500ms
        timeout: '200ms',
        // How often does an idle connection send a heartbeat?
        heart: '40s',
        // Timeout for waiting for heartbeat response
        heartTimeout: '1s',
      },
    },
    // This listener uses the connection provided by the 'listener portal' to provide services to the outside world.
    {
      network: 'tcp',
      addr: ':4000',
      dialer: {
        tag: 'portal',
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
