import { api, sql } from '@nitric/sdk'
import pg from 'pg'
const { Client } = pg

const myDb = sql('my-db')
const mySecondDb = sql('my-second-db')
const dbApi = api('my-db-api')

const getClient = async () => {
  const connStr = await myDb.connectionString()
  const client = new Client(connStr)

  return client
}

dbApi.get('/get', async (ctx) => {
  const client = await getClient()

  const res = await client.query('SELECT $1::text as message', ['Hello world!'])
  await client.end()

  return ctx.res.json(res.rows[0].message)
})
