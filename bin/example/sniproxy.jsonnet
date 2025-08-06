// This is an example of sniproxy dialer
{
  dialer: [
    // Set up a socks dialer for connecting to google
    {
      // Use this tag in listener to specify dialer
      tag: 'google',
      // Connection timeout
      timeout: '200ms',
      // Connection URL, optional parameters network and addr override the addr in the URL
      url: 'socks://10.89.0.1:1080',
      socks:{
        // user:'',
        // password:'',
        connect:'www.google.com:443',
      },
    },
    // Set up a direct dialer for connecting to baidu
    {
      // Use this tag in listener to specify dialer
      tag: 'baidu',
      // Connection timeout
      timeout: '200ms',
      // Connection URL, optional parameters network and addr override the addr in the URL
      url: 'basic://www.baidu.com:443',
    },
  ],
  sniproxy: [
    // This listener receives tcp connections
    {
      network: 'tcp',
      addr: ':443',
      // Sniff sni timeout, Default 500ms
      timeout:'500ms',
      router:[ 
        {
            matcher:[
                {
                    // 'equal' 'prefix' 'suffix' 'regexp' 
                    type: 'equal',
                    value: 'www.baidu.com',
                },
            ],
            dialer: {
                tag: 'baidu',
                close: '1s',
            },
        },
        {
            matcher:[
                {
                    // 'equal' 'prefix' 'suffix' 'regexp' 
                    type: 'equal',
                    value: 'www.google.com',
                },
            ],
            dialer: {
                tag: 'google',
                close: '1s',
            },
        },
      ]
    },
  ],
}
