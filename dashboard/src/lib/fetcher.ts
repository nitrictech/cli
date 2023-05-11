export const fetcher = (options?: RequestInit) => (url: string) =>
  fetch(url, options).then((r) => r.json());
