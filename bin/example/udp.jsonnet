// Here's a demonstration of how to set up udp over tcp

// This is the configuration on the server, which parses udp from tcp and transmits it to the destination service
local server = {
  dialer: [
    {
      tag: 'google-dns',
      timeout: '200ms',
      url: 'basic://8.8.8.8:53',
      network: 'udp',
      udp: {
        frame: 16,
        timeout: '60s',
        size: 1500,
      },
    },
  ],
  listener: [
    {
      network: 'pipe',
      addr: 'streamf/pipe.socket',
      dialer: {
        tag: 'google-dns',
        close: '1s',
      },
    },
  ],
};

// This is a reverse proxy. It receives udp, packages it, and transmits it to the server using tcp.
local proxy = {
  dialer: [
    {
      tag: 'udp-over-tcp',
      timeout: '200ms',
      url: 'basic://',
      network: 'pipe',
      addr: 'streamf/pipe.socket',
    },
  ],
  listener: [
    {
      network: 'udp',
      addr: ':4000',
      dialer: {
        tag: 'udp-over-tcp',
        close: '1s',
      },
      udp: {
        frame: 16,
        timeout: '60s',
        size: 1500,
      },
    },
  ],
};

{
  dialer: server.dialer + proxy.dialer,
  listener: server.listener + proxy.listener,
  logger: {
    source: true,
  },
}
