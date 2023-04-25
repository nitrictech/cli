import type { FieldRow } from "../components/shared/FieldRows";

export const sortedUniq = (arr: any[]) => [...new Set(arr)].sort();

export const uniqBy = (arr: any[], iteratee: any) => {
  if (typeof iteratee === "string") {
    const prop = iteratee;
    iteratee = (item: any) => item[prop];
  }

  return arr.filter(
    (x, i, self) => i === self.findIndex((y) => iteratee(x) === iteratee(y))
  );
};

export const getHost = () => {
  if (typeof window === "undefined") {
    return null;
  }

  return window && window.location.host.startsWith("127.0.0.1")
    ? "localhost:49152"
    : window.location.host;
};

export const fieldRowArrToHeaders = (arr: FieldRow[]) => {
  const headers = new Headers();
  arr.forEach((obj) => {
    if (obj.key) {
      headers.append(obj.key, obj.value);
    }
  });
  return headers;
};

export const headersToObject = (headers: Headers): Record<string, string> => {
  return Array.from(headers.entries()).reduce((acc, [key, value]) => {
    acc[key] = value;
    return acc;
  }, {} as Record<string, string>);
};
