export const copyToClipboard = (str: string) => {
  const focused = window.document.hasFocus()
  if (focused) {
    window.navigator?.clipboard?.writeText(str)
  } else {
    console.warn('Unable to copy to clipboard')
  }
}
