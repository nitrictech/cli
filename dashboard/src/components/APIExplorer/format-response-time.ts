export function formatResponseTime(milliseconds: number): string {
  if (milliseconds < 1000) {
    return milliseconds + " ms";
  } else if (milliseconds < 60 * 1000) {
    return Math.floor(milliseconds / 1000) + " s";
  } else if (milliseconds < 60 * 60 * 1000) {
    return Math.floor(milliseconds / (60 * 1000)) + " m";
  } else if (milliseconds < 24 * 60 * 60 * 1000) {
    return Math.floor(milliseconds / (60 * 60 * 1000)) + " h";
  } else {
    return Math.floor(milliseconds / (24 * 60 * 60 * 1000)) + " d";
  }
}
