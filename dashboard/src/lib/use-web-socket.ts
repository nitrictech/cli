import useSWRSubscription from "swr/subscription";
import type { WebSocketResponse } from "../types";
import { getHost } from "./utils";

export const useWebSocket = () => {
  const host = getHost();

  const { data, error } = useSWRSubscription(
    host ? `ws://${host}/ws` : null,
    (key, { next }) => {
      const socket = new WebSocket(key);
      socket.addEventListener("message", (event) => {
        const message = JSON.parse(event.data);

        next(null, message);
      });
      // @ts-ignore
      socket.addEventListener("error", (event) => next(event.error));
      return () => socket.close();
    }
  );

  return {
    data: data as WebSocketResponse | null,
    error,
  };
};
