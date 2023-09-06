export const getHost = () => {
  if (typeof window === "undefined") {
    return null;
  }

  return window && window.location.host.startsWith("localhost")
    ? "localhost:49152"
    : window.location.host;
};
