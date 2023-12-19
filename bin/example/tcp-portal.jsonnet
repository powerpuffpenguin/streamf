local bridge = {
  dialer: [
    {
      tag: 'tcp',
      timeout: '200ms',
      url: 'basic://example.com?addr=localhost:2000',
    },
  ],
  bridge: [
    {
      timeout: '200ms',
      url: 'basic://',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      dialer: {
        tag: 'tcp',
        close: '1s',
      },
    },
  ],
};
local portal = {
  dialer: [
    {
      tag: 'portal',
      timeout: '200ms',
      url: 'basic://',
      network: 'portal',
      addr: 'listener portal',
    },
  ],
  listener: [
    {
      tag: 'listener portal',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      mode: 'portal',
      portal: {
        // Wait connect timeout
        // Default 500ms
        timeout: '200ms',
        // If not true, send ack to confirm that the connection is valid.
        // Do not set this to true on network connections
        // noAck: true,
        // How often does an idle connection send a heartbeat?
        heart: '40s',
        // Timeout for waiting for heartbeat response
        heartTimeout: '1s',
      },
    },
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
