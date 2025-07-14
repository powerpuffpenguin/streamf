// This is an example of port forwarding
{
  // Set a basic forwarding target
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
      dialer: {
        // Forward to the dialer with tag 'tcp'
        tag: 'socks',
        // After one end of the connection is disconnected, wait for one second before closing the other end
        //  (waiting for untransmitted data to continue transmitting)
        close: '1s',
      },
    },
  ],
}
