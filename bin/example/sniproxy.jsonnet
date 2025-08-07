// This is an example of sniproxy dialer
{
  pool: {
    size: 1024 * 32,
    cache: 128,
  },
  logger: {
    level: 'info',
    source: false,
  },
  dialer: [
    // Set up a socks dialer for connecting to google
    {
      // Use this tag in listener to specify dialer
      tag: 'google',
      // Connection timeout
      timeout: '200ms',
      // Connection URL, optional parameters network and addr override the addr in the URL
      url: 'socks://10.89.0.1:1080',
      socks: {
        // user:'',
        // password:'',
        connect: 'www.google.com:443',
      },
    },
    // Set up a direct dialer for connecting to bing
    {
      // Use this tag in listener to specify dialer
      tag: 'bing',
      // Connection timeout
      timeout: '200ms',
      // Connection URL, optional parameters network and addr override the addr in the URL
      url: 'basic://www.bing.com:443',
    },
    {
      tag: 'default',
      timeout: '200ms',
      url: 'basic://192.168.251.50:8443',
    },
    {
      tag: 'fallback',
      timeout: '200ms',
      url: 'basic://192.168.251.50:10001',
    },
  ],
  sniproxy: [
    // This listener receives tcp connections
    {
      network: 'tcp',
      addr: ':443',
      // Sniff sni timeout, Default 500ms
      timeout: '500ms',
      // Optionally dialer. for no matching SNI
      default: {
        tag: 'default',
        close: '1s',
      },
      // Optionally dialer. for non-TLS/unknown-TLS
      fallback: {
        tag: 'fallback',
        close: '1s',
      },
      // sni matching routes
      router: [
        {
          matcher: [
            {
              // - 'accuracy' is matched first and must be unique. This is the default value and you don't need to explicitly define type
              // - 'prefix' or 'suffix' will match the first matched route in the order configured after 'accuracy'
              // - 'regexp' will be matched last
              type: 'accuracy',
              value: 'www.bing.com',
            },
          ],
          dialer: {
            tag: 'bing',
            close: '1s',
          },
        },
        {
          matcher: [
            {
              value: 'www.google.com',
            },
          ],
          dialer: {
            tag: 'google',
            close: '1s',
          },
        },
      ],
    },
  ],
}
