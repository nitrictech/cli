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

function generateTree(data: { Key: string }[]): FileData[] {
  const tree: FileData[] = [];

  data.forEach((item) => {
    const parts = item.Key.split("/");
    let parent = tree;
    let path = "";

    parts.forEach((part, index) => {
      const isDir = index < parts.length - 1;
      path += isDir ? `${part}/` : part;

      const existingDir = parent.find((node) => node.isDir && node.id === path);
      if (existingDir) {
        parent = existingDir.children!;
      } else {
        const newNode = {
          id: path,
          name: part,
          isDir,
          children: isDir ? [] : undefined,
        };
        parent.push(newNode);
        parent = newNode.children || [];
      }
    });
  });

  return tree;
}

function findNode(id: string, node: FileArray | FileData): FileData | null {
  if (Array.isArray(node)) {
    return findNode(id, { children: node } as any);
  }

  if (node.id === id) {
    return node;
  }

  if (node?.children) {
    for (const child of node.children) {
      const found = findNode(id, child);
      if (found !== null) {
        return found;
      }
    }
  }

  return null;
}

function getAllFiles(
  node: Partial<FileData>,
  files: Partial<FileData>[] = []
): FileData[] {
  if (node.isDir && node.children) {
    // if the current node is a directory, recursively process all children
    for (const child of node.children) {
      getAllFiles(child, files);
    }
  } else {
    // if the current node is a file, add it to the array of files
    files.push(node);
  }

  return files as FileData[];
}

const actionsToDisable: string[] = [
  ChonkyActions.SortFilesBySize.id,
  ChonkyActions.SortFilesByDate.id,
];

const FileBrowser: FC<Props> = ({ bucket }) => {
  const [rootFiles, setRootFiles] = useState<FileArray>([]);
  const [folderFiles, setFolderFiles] = useState<FileArray>([]);
  const [folderPrefix, setFolderPrefix] = useState<string>("/");
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
    [bucket, folderPrefix]
  );

  const getFilePath = (fileId: string) => `/${fileId}`;

  useEffect(() => {
    if (contents?.length) {
      const tree = generateTree(contents);
      setRootFiles(tree);
    } else {
      setRootFiles([]);
    }
  }, [contents]);

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

        setFolderPrefix(newPrefix);
      } else if (actionData.id === ChonkyActions.DeleteFiles.id) {
        const filesToDelete = getAllFiles({
          children: actionData.state.selectedFilesForAction,
          isDir: true,
        });

        // TODO perhaps add a confirm dialog?
        await Promise.all(
          filesToDelete.map((file) => deleteFile(getFilePath(file.id)))
        );

        const filesLeftCount = getAllFiles({
          children: folderFiles,
          isDir: true,
        }).length;

        // if no files left, simulate all being removed
        if (filesToDelete.length === filesLeftCount) {
          setFolderFiles([]);
        }

        mutate();
      }
    },
    [setFolderPrefix, bucket, contents, folderFiles]
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
              ? [currentPrefix, prefixPart].join("/")
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
  }, [folderPrefix, bucket, rootFiles]);

  useEffect(() => {
    if (folderPrefix === "/") {
      setFolderFiles(rootFiles);
    } else {
      const foundNode = findNode(folderPrefix, rootFiles);
      if (foundNode?.isDir) {
        setFolderFiles(foundNode.children);
      }
    }
  }, [folderPrefix, rootFiles]);

  return (
    <Loading className="my-20" delay={500} conditionToShow={!loading}>
      <div>
        <h2 className="mb-4">Bucket File Explorer</h2>
        <div style={{ height: 300 }} className="file-explorer">
          <ChonkFileBrowser
            instanceId={bucket}
            files={folderFiles}
            disableDefaultFileActions={actionsToDisable}
            fileActions={[ChonkyActions.DeleteFiles]}
            folderChain={folderChain}
            onFileAction={handleFileAction}
            thumbnailGenerator={(file) =>
              !file.isDir
                ? `${data?.storageAddress}/${bucket}${getFilePath(file.id)}`
                : null
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
