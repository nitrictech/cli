export const downloadFiles = async (
  files: Array<{ url: string; name: string }>,
): Promise<void> => {
  const promises = files.map(async (file) => {
    const response = await fetch(file.url)
    const blob = await response.blob()
    const link = document.createElement('a')
    link.href = window.URL.createObjectURL(blob)
    link.setAttribute('download', file.name)
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  })

  await Promise.all(promises)
}
