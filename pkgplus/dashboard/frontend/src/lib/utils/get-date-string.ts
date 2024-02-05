// Returns a user friendly time representation
export const getDateString = (requestTime: number) => {
  const currentDate = new Date()
  const requestDate = new Date(requestTime)

  const outputTimeString = (time: number, word: string) =>
    `${time} ${time > 1 ? word + 's' : word} ago`

  // Time diff is initially the difference in milliseconds, so convert to seconds
  const secondsDifference =
    (currentDate.getTime() - requestDate.getTime()) / 1000
  // Time is less than a minute
  if (secondsDifference < 60) {
    return 'just now'
  }
  // Time is less than an hour
  if (secondsDifference < 3600) {
    return outputTimeString(Math.floor(secondsDifference / 60), 'min')
  }
  // Time is less than a day
  if (secondsDifference < 86400) {
    return outputTimeString(Math.floor(secondsDifference / 3600), 'hour')
  }
  // Time is less than a week
  if (secondsDifference < 604800) {
    return outputTimeString(Math.floor(secondsDifference / 86400), 'day')
  }
  // Time is less than a month
  if (secondsDifference < 2630000) {
    return outputTimeString(Math.floor(secondsDifference / 604800), 'week')
  }
  // Time is less than a year
  if (secondsDifference < 31536000) {
    return outputTimeString(Math.floor(secondsDifference / 2630000), 'month')
  }
  // Time is greater than a year
  return outputTimeString(Math.floor(secondsDifference / 31536000), 'year')
}
