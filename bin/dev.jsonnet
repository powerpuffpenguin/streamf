{
  logger: {
    level: 'debug',
    // source: true,
  },
  listener: [
    {
      network: 'unix',
      address: '@cf.socket',
      close: '1s',
    },
  ],
}
