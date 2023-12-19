// Here the portal/bridge are set together to facilitate testing. The real environment is to set the 'bridge' separately to the intranet server that needs to be mapped.
local bridge = {
  dialer: [
    // Services to map
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
  ],
  bridge: [
    // This 'bridge' will connect the 'portal'
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
// Here the portal/bridge are set together to facilitate testing. The real environment is to set the 'portal' separately to a server with a public network IP.
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
    // This listener uses 'portal' mode to receive connections from intranet mappings
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
