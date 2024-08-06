import { api, secret } from '@nitric/sdk'

const mySecret = secret('my-secret').allow('access', 'put')

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
