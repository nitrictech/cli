import { useCallback } from "react";
import useSWR from "swr";
import { fetcher } from "./fetcher";
import type { BucketFile } from "../types";
import { STORAGE_API } from "./const";

export const useBucket = (bucket?: string, prefix?: string) => {
  const { data, mutate } = useSWR<BucketFile[]>(
    bucket && prefix
      ? `${STORAGE_API}?action=list-files&bucket=${bucket}`
      : null,
    fetcher()
  );

  const writeFile = useCallback(
    async (file: File) => {
      return fetch(
        `${STORAGE_API}?action=write-file&bucket=${bucket}&fileKey=${encodeURI(
          file.name
        )}`,
        {
          method: "PUT",
          body: file,
        }
      );
    },
    [bucket]
  );

  const deleteFile = useCallback(
    (fileKey: string) => {
      return fetch(
        `${STORAGE_API}?action=delete-file&bucket=${bucket}&fileKey=${encodeURI(
          fileKey
        )}`,
        {
          method: "DELETE",
        }
      );
    },
    [bucket]
  );

  return {
    data,
    mutate,
    deleteFile,
    writeFile,
    loading: !data,
  };
};
