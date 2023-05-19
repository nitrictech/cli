import { getFileExtension } from "../../lib/get-file-extension";
import type { APIResponse } from "../../types";
import CodeEditor from "./CodeEditor";

interface Props {
  response: APIResponse;
}

const APIResponseContent: React.FC<Props> = ({ response }) => {
  const contentType = response.headers!["content-type"];

  if (contentType.startsWith("image/")) {
    return (
      <img
        data-testid="response-image"
        src={response.data}
        alt={"response content"}
        className="w-full max-h-96 object-contain"
      />
    );
  } else if (contentType.startsWith("video/")) {
    return <video src={response.data} controls />;
  } else if (contentType.startsWith("audio/")) {
    return <audio src={response.data} controls />;
  } else if (contentType === "application/pdf") {
    return <iframe title="Response PDF" className="h-96" src={response.data} />;
  } else if (
    contentType.startsWith("application/") &&
    contentType !== "application/json"
  ) {
    const ext = getFileExtension(contentType);

    const fileName = response.data.split("/")[3] + ext;

    return (
      <div className="my-4">
        The response is binary, you can{" "}
        <a
          href={response.data}
          data-testid="response-binary-link"
          className="underline"
          download={fileName}
        >
          download the file here
        </a>
        .
      </div>
    );
  }

  return (
    <CodeEditor
      id="api-response"
      enableCopy
      contentType={contentType}
      value={response.data}
      readOnly
    />
  );
};

export default APIResponseContent;