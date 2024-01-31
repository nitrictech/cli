export const isValidUrl = (value: string) => {
  try {
    return !decodeURI(value).includes('%')
  } catch (e) {
    return false
  }
}
