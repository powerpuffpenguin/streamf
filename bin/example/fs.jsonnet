local auth = [
  {
    username: 'dev',
    password: '123',
  },
];
{
  listener: [
    {
      network: 'tcp',
      addr: ':4000',
      mode: 'http',
      router: [
        {
          method: 'FS',
          pattern: '/fs',
          fs: '/tmp',
          auth: auth,
        },
        {
          method: 'API',
          pattern: '/',
          auth: auth,
        },
      ],
    },
  ],
}
