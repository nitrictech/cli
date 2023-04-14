import type { APIResponse } from "../../types";
import CodeEditor from "./CodeEditor";

interface Props {
  response: APIResponse;
}

const APIResponseContent: React.FC<Props> = ({ response }) => {
  const contentType = response.headers!["content-type"];

  if (contentType.startsWith("image/")) {
    return (
      <img src={response.data} className='w-full max-h-96 object-contain' />
    );
  } else if (contentType.startsWith("video/")) {
    return <video src={response.data} controls />;
  } else if (contentType.startsWith("audio/")) {
    return <audio src={response.data} controls />;
  } else if (contentType === "application/pdf") {
    return <iframe className='h-96' src={response.data} />;
  } else if (
    contentType.startsWith("application/") &&
    contentType !== "application/json"
  ) {
    return (
      <div className='my-4'>
        The response is binary, you can{" "}
        <a href={response.data} className='underline'>
          download the file here
        </a>
        .
      </div>
    );
  }

  return (
    <CodeEditor contentType={contentType} value={response.data} readOnly />
  );
};

export default APIResponseContent;
