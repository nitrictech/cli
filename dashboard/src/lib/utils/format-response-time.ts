export function formatResponseTime(milliseconds: number): string {
  if (milliseconds < 1000) {
    return milliseconds.toFixed(2) + " ms";
  } else if (milliseconds < 60 * 1000) {
    return (milliseconds / 1000).toFixed(2) + " s";
  } else if (milliseconds < 60 * 60 * 1000) {
    return (milliseconds / (60 * 1000)).toFixed(2) + " m";
  } else if (milliseconds < 24 * 60 * 60 * 1000) {
    return (milliseconds / (60 * 60 * 1000)).toFixed(2) + " h";
  } else {
    return (milliseconds / (24 * 60 * 60 * 1000)).toFixed(2) + " d";
  }
}
