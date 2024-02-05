export const sortedUniq = (arr: any[]) => [...new Set(arr)].sort()

export const uniqBy = (arr: any[], iteratee: any) => {
  if (typeof iteratee === 'string') {
    const prop = iteratee
    iteratee = (item: any) => item[prop]
  }

  return arr.filter(
    (x, i, self) => i === self.findIndex((y) => iteratee(x) === iteratee(y)),
  )
}
