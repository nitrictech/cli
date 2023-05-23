import useSWRSubscription from "swr/subscription";
import type { WebSocketResponse } from "../../types";
import { getHost } from "../utils";
import { toast } from "react-hot-toast";
import { useRef } from "react";
import { isEqual } from "radash";

export const useWebSocket = () => {
  const toastIdRef = useRef<string>();
  const prevDataRef = useRef<WebSocketResponse>();
  const timeoutIdRef = useRef<NodeJS.Timeout>();
  const host = getHost();

  const showSuccessMessage = () => {
    if (toastIdRef.current) {
      toast.success("Refreshed", {
        id: toastIdRef.current,
      });
      toastIdRef.current = "";
    }
  };

  const { data, error } = useSWRSubscription(
    host ? `ws://${host}/ws` : null,
    (key, { next }) => {
      const socket = new WebSocket(key);

      socket.addEventListener("message", (event) => {
        const message = JSON.parse(event.data) as WebSocketResponse;

        // must have previous data to show refresh
        if (prevDataRef.current) {
          // if no toast showing, show refreshing loader
          if (!toastIdRef.current) {
            toastIdRef.current = toast.loading("Refreshing");

            timeoutIdRef.current = setTimeout(() => showSuccessMessage(), 3500);
          } else if (isEqual(prevDataRef.current, message)) {
            // this block is for multiple messages, clear any pending timeouts
            if (timeoutIdRef.current) {
              clearTimeout(timeoutIdRef.current);
            }

            timeoutIdRef.current = setTimeout(() => showSuccessMessage(), 1000);
          }
        }

        prevDataRef.current = message;

        next(null, message);
      });

      socket.addEventListener("error", (event: any) => next(event.error));
      return () => socket.close();
    }
  );

  return {
    data: data as WebSocketResponse | null,
    error,
    loading: !data,
  };
};
