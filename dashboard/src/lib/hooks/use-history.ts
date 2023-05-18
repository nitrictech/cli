import { useCallback } from "react";
import useSWR from "swr";
import { fetcher } from "./fetcher";
import type { BucketFile, HistoryItem } from "../../types";
import { HISTORY_API } from "../constants";

export function useHistory<T extends HistoryItem>(recordType: string) {
  const { data, mutate } = useSWR<T[]>(
    recordType ? `${HISTORY_API}?type=${recordType}` : null,
    fetcher()
  );

  const deleteHistory = useCallback(() => {
    return fetch(`${HISTORY_API}?type=${recordType}`, {
      method: "DELETE",
    });
  }, [recordType]);

  return {
    data,
    mutate,
    deleteHistory,
    loading: !data,
  };
}
