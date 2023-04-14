export function formatFileSize(bytes: number): string {
  const units = ["bytes", "KB", "MB", "GB", "TB"];
  let size = bytes;
  let unitIndex = 0;

  while (size >= 1024 && unitIndex < units.length - 1) {
    size /= 1024;
    unitIndex++;
  }

  const unit = units[unitIndex];

  return `${bytes > 1024 ? size.toFixed(2) : size} ${unit}`;
}
