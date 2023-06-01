import { useCallback } from "react";
import type { History } from "../../types";
import { getHost } from "../utils";
import useSWRSubscription from "swr/subscription";
import toast from "react-hot-toast";

export function useHistory(recordType: string) {
  const host = getHost();

  const { data, error } = useSWRSubscription(
    host ? `ws://${host}/history` : null,
    (key, { next }) => {
      const socket = new WebSocket(key);

      socket.addEventListener("message", (event) => {
        const message = JSON.parse(event.data) as History;

        next(null, message);
      });

      socket.addEventListener("error", (event: any) => next(event.error));
      return () => socket.close();
    }
  );

  const deleteHistory = useCallback(async () => {
    const resp = await fetch(`http://${host}/api/history?type=${recordType}`, {
      method: "DELETE",
    });

    if (resp.ok) {
      toast.success("Cleared History");
    }
  }, [recordType]);

  if (error) {
    console.error(error);
  }

  return {
    data: data as History | null,
    deleteHistory,
    loading: !data,
  };
}
