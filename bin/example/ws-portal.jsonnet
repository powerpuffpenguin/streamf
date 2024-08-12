
local bridge_name(s) = "bridge."+s;
local bridge={
    dialer:[
        {
            tag: bridge_name('connect_tcp'),
            timeout: '200ms',
            url: 'basic://127.0.0.1:1234',
        },
        {
            tag: bridge_name('connect_h2c'),
            timeout: '200ms',
            url: 'basic://',
            network: 'pipe',
            addr: 'pipe.h2c',
        },
    ],
    listener:[
        {
            network: 'pipe',
            addr: 'pipe.h2c',
            mode: 'http',
            router: [
                {
                    method: 'POST',
                    pattern: '/portal',
                    dialer: {
                        tag: bridge_name('connect_tcp'),
                        close: '1s',
                    },
                },
            ]
        },
    ],
    bridge:[
        {
            timeout: '400ms',
            url: 'ws://video.com/video/ws',
            fast: true,
            network: 'tcp',
            addr: '127.0.0.1:9000',
            dialer: {
                tag: bridge_name('connect_h2c'),
                close: '1s',
            },
        },
    ],
};
local portal_name(s) = "portal."+s;
local portal={
    dialer:[
        {
            tag: portal_name('connect_h2c'),
            timeout: '1s',
            url: 'http://abc.com/portal',
            network: 'portal',
            addr:portal_name( 'portal_h2c'),
            method: 'POST',
            retry: 2,
        },
    ],
    listener:[
        {
            network: 'tcp',
            addr: ':9000',
            mode: 'http',
            router: [
                 {
                    method: 'API',
                    pattern: '/api',
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
                    fast: true,
                    portal: {
                        tag:portal_name( 'portal_h2c'),
                        timeout: '1s',
                        heart: '40s',
                        heartTimeout: '1s',
                    },
                },
            ],
        },
        {
            network: 'tcp',
            addr: ':2000',
            dialer: {
                tag: portal_name('connect_h2c'),
                close: '1s',
            },
        },
    ],
};
{
  dialer: bridge.dialer + portal.dialer,
  listener: bridge.listener + portal.listener,
  bridge: bridge.bridge,
  logger:{
    level:"error"
  },
}
