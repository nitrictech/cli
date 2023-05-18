export const getHost = () => {
  if (typeof window === "undefined") {
    return null;
  }

  return window && window.location.host.startsWith("127.0.0.1")
    ? "localhost:49152"
    : window.location.host;
};
