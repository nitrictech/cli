import { formatJSON, headersToObject } from '../utils'

export const generateResponse = async (res: Response, startTime: number) => {
  const contentType = res.headers.get('Content-Type')

  let data

  if (contentType === 'application/json') {
    data = formatJSON(await res.text())
  } else if (
    contentType?.startsWith('image/') ||
    contentType?.startsWith('video/') ||
    contentType?.startsWith('audio/') ||
    contentType?.startsWith('application')
  ) {
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)

    data = url
  } else {
    data = await res.text()
  }

  const endTime = window.performance.now()
  const responseSize = res.headers.get('Content-Length')

  return {
    data,
    time: endTime - startTime,
    status: res.status,
    size: responseSize ? parseInt(responseSize) : 0,
    headers: headersToObject(res.headers),
  }
}
