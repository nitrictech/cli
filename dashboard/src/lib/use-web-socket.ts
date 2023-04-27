import useSWRSubscription from "swr/subscription";
import type { WebSocketResponse } from "../types";
import { getHost } from "./utils";
import { toast } from "react-hot-toast";
import { useRef } from "react";
import { isEqual } from "radash";

export const useWebSocket = () => {
  const toastIdRef = useRef<string>();
  const prevDataRef = useRef<WebSocketResponse>();
  const host = getHost();

  const { data, error } = useSWRSubscription(
    host ? `ws://${host}/ws` : null,
    (key, { next }) => {
      const socket = new WebSocket(key);

      socket.addEventListener("message", (event) => {
        const message = JSON.parse(event.data) as WebSocketResponse;

        if (
          prevDataRef.current?.apis.length === message.apis.length &&
          !isEqual(prevDataRef.current, message) &&
          !toastIdRef.current
        ) {
          toastIdRef.current = toast.loading("Refreshing");
        } else if (
          toastIdRef.current &&
          prevDataRef.current?.apis.length === message.apis.length &&
          isEqual(prevDataRef.current, message)
        ) {
          toast.success("Refreshed", {
            id: toastIdRef.current,
          });
          toastIdRef.current = "";
        }

        next(null, message);

        prevDataRef.current = message;
      });

      // @ts-ignore
      socket.addEventListener("error", (event) => next(event.error));
      return () => socket.close();
    }
  );

  //const [debouncedData] = useDebounce(data, 2500);

  // useEffect(() => {
  //   if (toastIdRef.current) {
  //     toast.success("Refreshed", {
  //       id: toastIdRef.current,
  //     });

  //     setTimeout(() => {
  //       toastIdRef.current = "";
  //     }, 500);
  //   }
  // }, [debouncedData]);

  return {
    data: data as WebSocketResponse | null,
    error,
    loading: !data,
  };
};
