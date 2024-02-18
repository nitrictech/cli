export function getFileExtension(contentType: string): string {
  switch (contentType) {
    case 'application/xml':
    case 'text/xml':
      return '.xml'
    case 'application/json':
      return '.json'
    case 'application/pdf':
      return '.pdf'
    case 'application/zip':
      return '.zip'
    case 'application/octet-stream':
      return '.bin'
    default:
      return ''
  }
}
