import { api, bucket, kv, http, schedule, topic, websocket } from '@nitric/sdk'

const firstApi = api('first-api')
const secondApi = api('second-api')

const myBucket = bucket('test-bucket').for('reading', 'writing', 'deleting')

myBucket.on('delete', '*', (ctx) => {
  console.log(ctx)
})

const socket = websocket('socket')

const socket2 = websocket('socket-2')

const socket3 = websocket('socket-3')

const connections = kv<Record<string, any>>('connections').for(
  'getting',
  'setting',
  'deleting',
)
interface Doc {
  firstCount: number
  secondCount: number
}

const col = kv<Doc>('test-collection').for('getting', 'setting', 'deleting')

firstApi.get('/schedule-count', async (ctx) => {
  try {
    const data = await col.get('schedule-count')

    return ctx.res.json(data)
  } catch (e) {
    return ctx.res.json({
      firstCount: 0,
      secondCount: 0,
    } as Doc)
  }
})

firstApi.get('/topic-count', async (ctx) => {
  try {
    const data = await col.get('topic-count')

    return ctx.res.json(data)
  } catch (e) {
    return ctx.res.json({
      firstCount: 0,
      secondCount: 0,
    } as Doc)
  }
})

// test all methods
firstApi.post('/all-methods', async (ctx) => ctx)
firstApi.put('/all-methods', async (ctx) => ctx)
firstApi.patch('/all-methods', async (ctx) => ctx)
firstApi.get('/all-methods', async (ctx) => ctx)
firstApi.delete('/all-methods', async (ctx) => ctx)
firstApi.options('/all-methods', async (ctx) => ctx)

firstApi.get('/path-test/:name', async (ctx) => {
  const { name } = ctx.req.params

  ctx.res.body = `Hello ${name}`

  return ctx
})

firstApi.get('/header-test', async (ctx) => {
  const data = ctx.req.headers

  return ctx.res.json({
    headers: data,
  })
})

firstApi.get('/query-test', async (ctx) => {
  const data = ctx.req.query

  return ctx.res.json({
    queryParams: data,
  })
})

firstApi.post('/json-test', async (ctx) => {
  const data = ctx.req.json()

  return ctx.res.json({
    requestData: data,
  })
})

secondApi.get('/content-type-html', (ctx) => {
  const html = `
    <html>
      <head>
        <title>My Web Page</title>
      </head>
      <body>
        <h1>Welcome to my web page</h1>
        <p>This is some sample HTML content.</p>
      </body>
    </html>
  `
  ctx.res.headers = { 'content-type': ['text/html'] }
  ctx.res.body = html.trim()

  return ctx
})

secondApi.get('/content-type-css', (ctx) => {
  const css = `
    body {
      font-family: Arial, sans-serif;
      background-color: #f1f1f1;
    }

    h1 {
      color: blue;
    }

    p {
      color: green;
    }
  `
  ctx.res.headers = { 'content-type': ['text/css'] }
  ctx.res.body = css.trim()

  return ctx
})

secondApi.get('/content-type-image', (ctx) => {
  const svgData = `
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
      <rect width="100%" height="100%" fill="#ff0000"/>
      <text x="50%" y="50%" font-size="48" fill="#ffffff" text-anchor="middle">SVG</text>
    </svg>
  `
  ctx.res.headers = { 'content-type': ['image/svg+xml'] }
  ctx.res.body = svgData

  return ctx
})

secondApi.get('/content-type-xml', (ctx) => {
  const xmlData = `
    <?xml version="1.0" encoding="UTF-8"?>
    <data>
      <user>
        <name>John Doe</name>
        <email>john.doe@example.com</email>
      </user>
      <user>
        <name>Jane Smith</name>
        <email>jane.smith@example.com</email>
      </user>
    </data>
  `
  ctx.res.headers = { 'content-type': ['text/xml'] }
  ctx.res.body = xmlData.trim()

  return ctx
})

secondApi.get('/content-type-binary', (ctx) => {
  const xmlData = `
    <?xml version="1.0" encoding="UTF-8"?>
    <data>
      <user>
        <name>John Doe</name>
        <email>john.doe@example.com</email>
      </user>
      <user>
        <name>Jane Smith</name>
        <email>jane.smith@example.com</email>
      </user>
    </data>
  `
  ctx.res.headers = { 'content-type': ['application/xml'] }
  ctx.res.body = xmlData.trim()

  return ctx
})

secondApi.get('/image-from-bucket', async (ctx) => {
  const image = await myBucket.file('images/photo.jpg').read()

  ctx.res.body = image
  ctx.res.headers = { 'Content-Type': ['image/jpeg'] }
  return ctx
})

secondApi.put('/image-from-bucket', async (ctx) => {
  const imageData = ctx.req.data

  await myBucket.file('images/photo.jpg').write(imageData)

  return ctx
})

secondApi.put('/very-nested-files', async (ctx) => {
  const { fileName } = ctx.req.query
  const data = ctx.req.data

  await myBucket.file(`5/4/3/2/1/${fileName}`).write(data)

  return ctx
})

secondApi.delete('/image-from-bucket', async (ctx) => {
  await myBucket.file('images/photo.jpg').delete()

  return ctx
})

schedule('process-tests').every('5 minutes', async (ctx) => {
  try {
    const data = await col.get('schedule-count')

    await col.set('schedule-count', {
      ...data,
      firstCount: data.firstCount + 1,
    })
  } catch (e) {
    await col.set('schedule-count', {
      firstCount: 1,
      secondCount: 0,
    })
  }
})

schedule('process-tests-2').every('5 minutes', async (ctx) => {
  try {
    const data = await col.get('schedule-count')

    await col.set('schedule-count', {
      ...data,
      secondCount: data.secondCount + 1,
    })
  } catch (e) {
    await col.set('schedule-count', {
      firstCount: 0,
      secondCount: 1,
    })
  }
})

topic('subscribe-tests').subscribe(async (ctx) => {
  try {
    const data = await col.get('topic-count')

    await col.set('topic-count', {
      ...data,
      firstCount: data.firstCount + 1,
    })
  } catch (e) {
    await col.set('topic-count', {
      firstCount: 1,
      secondCount: 0,
    })
  }
})

topic('subscribe-tests-2').subscribe(async (ctx) => {
  try {
    const data = await col.get('topic-count')

    await col.set('topic-count', {
      ...data,
      secondCount: data.secondCount + 1,
    })
  } catch (e) {
    await col.set('topic-count', {
      firstCount: 0,
      secondCount: 1,
    })
  }
})

// web sockets
socket.on('connect', async (ctx) => {
  let connectionsInfo = {}

  try {
    connectionsInfo = await connections.get('connections')
  } catch (e) {
    console.log(e)
  }

  await connections.set('connections', {
    ...(connectionsInfo || {}),
    [ctx.req.connectionId]: {},
  })
})

const deleteConnection = async (connectionId: string) => {
  const connectionsObj = await connections.get('connections')

  console.log(connectionsObj)

  delete connectionsObj[connectionId]

  console.log(connectionsObj)

  await connections.set('connections', connectionsObj)
}

const broadcast = async (data: string | Uint8Array) => {
  try {
    const connectionsObj = await connections.get('connections')

    await Promise.all(
      Object.keys(connectionsObj).map(async (connectionId) => {
        try {
          // will replace data with a strinified version of query if it exists (for tests)
          await socket.send(connectionId, data)
        } catch (e) {
          if (e.message.startsWith('13 INTERNAL: could not get connection')) {
            await deleteConnection(connectionId)
          }
        }
      }),
    )
  } catch (e) {}
}

socket.on('disconnect', async (ctx) => {
  await deleteConnection(ctx.req.connectionId)
})

socket.on('message', async (ctx) => {
  // broadcast message to all clients (including the sender)
  await broadcast(ctx.req.data)
})

socket2.on('connect', async (ctx) => {
  console.log('TODO')
})

socket2.on('message', (ctx) => {
  console.log(`Message: ${ctx.req.data}`)
})

socket3.on('connect', (ctx) => {
  ctx.res.success = false
})

socket3.on('disconnect', (ctx) => {
  ctx.res.success = false
})

socket3.on('message', (ctx) => {
  ctx.res.success = false
})
