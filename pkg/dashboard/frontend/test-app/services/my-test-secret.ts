import { api, secret } from '@nitric/sdk'

const mySecret = secret('my-first-secret').allow('access', 'put')

const mySecondSecret = secret('my-second-secret').allow('access', 'put')

const shhApi = api('my-secret-api')

shhApi.get('/get', async (ctx) => {
  const latestValue = await mySecret.latest().access()

  ctx.res.body = latestValue.asString()

  return ctx
})

shhApi.post('/set', async (ctx) => {
  const data = ctx.req.json()

  await mySecret.put(JSON.stringify(data))

  return ctx
})

shhApi.post('/set-binary', async (ctx) => {
  // Example usage
  const exampleArray = new Uint8Array([
    0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b,
    0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
    0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f, 0x20, 0x21, 0x22, 0x23,
    0x24, 0x25, 0x26, 0x27, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, 0x2e, 0x2f,
    // ... (add more bytes as needed)
  ])
  await mySecret.put(exampleArray)

  return ctx
})
