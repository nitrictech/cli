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
  const data = new Uint8Array(1024)
  for (let i = 0; i < data.length; i++) {
    data[i] = i % 256
  }

  await mySecret.put(data)

  return ctx
})
