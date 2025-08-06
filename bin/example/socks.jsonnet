// This is an example of socks5 dialer
{
  // Set socks5 dialer
  dialer: [
    {
      // Use this tag in listener to specify dialer
      tag: 'socks',
      // Connection timeout
      timeout: '200ms',
      // Connection URL, optional parameters network and addr override the addr in the URL
      url: 'socks://127.0.0.1:1081',
      socks:{
        // user:'',
        // password:'',
        connect:'10.89.0.1:2004',
      },
    },
  ],
  listener: [
    // This listener receives tcp connections
    {
      network: 'tcp',
      addr: ':4000',
      router: [
        {
          tag: 'socks',
          close: '1s',
        },
      ],
    },
  ],
}
