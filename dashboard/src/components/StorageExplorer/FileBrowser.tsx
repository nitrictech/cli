import {
  FileBrowser as ChonkFileBrowser,
  FileNavbar,
  FileToolbar,
  FileList,
  setChonkyDefaults,
  FileArray,
  ChonkyActions,
  ChonkyFileActionData,
  FileData,
  FileContextMenu,
} from "chonky";
import { FC, useCallback, useEffect, useMemo, useState } from "react";
import { useWebSocket } from "../../lib/use-web-socket";

import { ChonkyIconFA } from "chonky-icon-fontawesome";
import { downloadFiles } from "./download-files";
import { useBucket } from "../../lib/use-bucket";
import "./file-browser-styles.css";
import FileUpload from "./FileUpload";
import Loading from "../shared/Loading";
interface Props {
  bucket: string;
}

setChonkyDefaults({
  iconComponent: ChonkyIconFA,
});

function joinPath(...segments: any[]) {
  return segments.reduce((url, segment) => new URL(segment, url).pathname, "");
}

const actionsToDisable: string[] = [
  ChonkyActions.SortFilesBySize.id,
  ChonkyActions.SortFilesByDate.id,
];

const FileBrowser: FC<Props> = ({ bucket }) => {
  const [files, setFiles] = useState<FileArray>([]);
  const [folderPrefix, setKeyPrefix] = useState<string>("/");
  const { data } = useWebSocket();
  const {
    data: contents,
    writeFile,
    mutate,
    deleteFile,
    loading,
  } = useBucket(bucket, folderPrefix);

  const onDrop = useCallback(
    async (acceptedFiles: File[]) => {
      await Promise.all(acceptedFiles.map((file) => writeFile(file)));
      mutate();
    },
    [bucket]
  );

  const getFilePath = (fileId: string) =>
    `${folderChain.map((f) => f?.id).join("/")}${fileId}`;

  useEffect(() => {
    if (contents?.length) {
      setFiles(
        contents.map(
          (content) =>
            ({
              id: content.Key,
              name: content.Key!.split("/").pop() || "",
            } as FileData)
        )
      );
    } else {
      setFiles([]);
    }
  }, [folderPrefix, contents]);

  const handleFileAction = useCallback(
    async (actionData: ChonkyFileActionData) => {
      if (actionData.id === ChonkyActions.OpenFiles.id) {
        if (actionData.payload.files && actionData.payload.files.length !== 1)
          return;
        if (
          !actionData.payload.targetFile ||
          !actionData.payload.targetFile.isDir
        )
          return;

        const newPrefix = `${actionData.payload.targetFile.id.replace(
          /\/*$/,
          ""
        )}/`;

        setKeyPrefix(newPrefix);
      } else if (actionData.id === ChonkyActions.DeleteFiles.id) {
        await Promise.all(
          actionData.state.selectedFilesForAction.map((file) =>
            deleteFile(getFilePath(file.id))
          )
        );

        mutate();
      } else if (actionData.id === ChonkyActions.DownloadFiles.id) {
        downloadFiles(
          actionData.state.selectedFilesForAction.map((file) => ({
            name: file.name,
            url: `${data?.storageAddress}/${bucket}/${getFilePath(file.id)}`,
          }))
        );
      }
    },
    [setKeyPrefix, bucket]
  );

  const folderChain = useMemo(() => {
    if (bucket) {
      let folderChain: FileArray;
      if (folderPrefix === "/") {
        folderChain = [];
      } else {
        let currentPrefix = "";
        folderChain = folderPrefix
          .replace(/\/*$/, "")
          .split("/")
          .map((prefixPart): FileData => {
            currentPrefix = currentPrefix
              ? joinPath(currentPrefix, prefixPart)
              : prefixPart;
            return {
              id: currentPrefix,
              name: prefixPart,
              isDir: true,
            };
          });
      }
      folderChain.unshift({
        id: "/",
        name: bucket,
        isDir: true,
      });
      return folderChain;
    }

    return [];
  }, [folderPrefix, bucket]);

  return (
    <Loading className="my-20" delay={500} conditionToShow={!loading}>
      <div>
        <h2 className="mb-4">Bucket File Explorer</h2>
        <div style={{ height: 300 }} className="file-explorer">
          <ChonkFileBrowser
            instanceId={bucket}
            files={files}
            disableDefaultFileActions={actionsToDisable}
            fileActions={[
              ChonkyActions.DownloadFiles,
              ChonkyActions.DeleteFiles,
            ]}
            folderChain={folderChain}
            onFileAction={handleFileAction}
            thumbnailGenerator={(file) =>
              `${data?.storageAddress}/${bucket}${getFilePath(file.id)}`
            }
          >
            <FileNavbar />
            <FileToolbar />
            <FileList />
            <FileContextMenu />
          </ChonkFileBrowser>
        </div>
      </div>
      <div>
        <h2 className="mb-4">Upload Files</h2>
        <FileUpload multiple onDrop={onDrop} />
      </div>
    </Loading>
  );
};

export default FileBrowser;
