const isDev = import.meta.env.DEV;

export const getHost = () => {
  if (typeof window === "undefined") {
    return null;
  }

  return isDev ? "localhost:49152" : window.location.host;
};
