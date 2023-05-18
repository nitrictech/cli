import classNames from "classnames";
import { ArrowUpOnSquareIcon } from "@heroicons/react/24/outline";
import { DropzoneOptions, useDropzone } from "react-dropzone";
import type { FC } from "react";

type Props = DropzoneOptions;

const FileUpload: FC<Props> = ({ multiple, ...rest }) => {
  const { getRootProps, getInputProps, isDragActive } = useDropzone({
    multiple,
    ...rest,
  });

  return (
    <div
      {...getRootProps()}
      className={classNames(
        "relative cursor-pointer block w-full rounded-lg border-2 border-dashed border-gray-300 p-12 text-center hover:border-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2",
        isDragActive ? "border-blue-500" : ""
      )}
    >
      <ArrowUpOnSquareIcon className="mx-auto h-12 w-12 text-gray-400" />
      <input data-testid="file-upload" {...getInputProps()} />
      <p className="mt-2 block text-sm font-semibold text-gray-900">
        {isDragActive
          ? `Drop the ${multiple ? "files" : "file"} here ...`
          : `Drag or click to add ${multiple ? "files" : "file"}.`}
      </p>
    </div>
  );
};

export default FileUpload;
