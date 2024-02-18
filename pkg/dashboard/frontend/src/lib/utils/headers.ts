import type { FieldRow } from '../../components/shared/FieldRows'

export const headersToObject = (headers: Headers): Record<string, string> => {
  return Array.from(headers.entries()).reduce(
    (acc, [key, value]) => {
      acc[key] = value
      return acc
    },
    {} as Record<string, string>,
  )
}

export const fieldRowArrToHeaders = (arr: FieldRow[]) => {
  const headers = new Headers()
  arr.forEach((obj) => {
    if (obj.key) {
      headers.append(obj.key, obj.value)
    }
  })
  return headers
}
