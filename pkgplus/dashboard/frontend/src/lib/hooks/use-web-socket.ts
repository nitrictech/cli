import useSWRSubscription from "swr/subscription";
import type { WebSocketResponse } from "../../types";
import { getHost } from "../utils";
import { toast } from "react-hot-toast";
import { useRef, useState } from "react";
import { isEqual } from "radash";

export const useWebSocket = () => {
  const toastIdRef = useRef<string>();
  const prevDataRef = useRef<WebSocketResponse>();
  const timeoutIdRef = useRef<NodeJS.Timeout>();
  const socketRef = useRef<WebSocket>();
  const [state, setState] = useState<"open" | "error">();
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
      const connectWebSocket = () => {
        socketRef.current = new WebSocket(key);

        socketRef.current.addEventListener("open", () => setState("open"));

        socketRef.current.addEventListener("message", (event) => {
          const message = JSON.parse(event.data) as WebSocketResponse;

          // must have previous data to show refresh
          if (prevDataRef.current) {
            // if no toast showing, show refreshing loader
            if (!toastIdRef.current) {
              toastIdRef.current = toast.loading("Refreshing");

              timeoutIdRef.current = setTimeout(
                () => showSuccessMessage(),
                3500
              );
            } else if (isEqual(prevDataRef.current, message)) {
              // this block is for multiple messages, clear any pending timeouts
              if (timeoutIdRef.current) {
                clearTimeout(timeoutIdRef.current);
              }

              timeoutIdRef.current = setTimeout(
                () => showSuccessMessage(),
                1000
              );
            }
          }

          prevDataRef.current = message;

          next(null, message);
        });

        socketRef.current.addEventListener("close", () => {
          // Retry WebSocket connection after a delay
          timeoutIdRef.current = setTimeout(() => {
            connectWebSocket(); // Reconnect WebSocket
          }, 1500); // Adjust the delay as needed
        });

        socketRef.current.addEventListener("error", (event: any) => {
          setState("error");
          next(event.error);
        });
      };

      connectWebSocket();

      return () => {
        socketRef.current?.close();

        if (timeoutIdRef.current) {
          clearTimeout(timeoutIdRef.current);
        }
      };
    }
  );

  return {
    data: data as WebSocketResponse | null,
    error,
    loading: !data,
    state,
  };
};
