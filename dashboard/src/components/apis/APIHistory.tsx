import type { ApiHistoryItem, Endpoint } from "../../types";
import Badge from "../shared/Badge";
import { formatJSON, getDateString } from "../../lib/utils";
import { Disclosure } from "@headlessui/react";
import { ChevronUpIcon } from "@heroicons/react/20/solid";
import { useState } from "react";
import { Tabs } from "../shared";
import CodeEditor from "./CodeEditor";
import APIResponseContent from "./APIResponseContent";
import TableGroup from "../shared/TableGroup";

interface Props {
  history: ApiHistoryItem[];
  selectedRequest: {
    method: string;
    path: string;
  };
}

const checkEquivalentPaths = (matcher: string, path: string): boolean => {
  // If the paths are equal regardless of query params
  if (path.split("?").length > 1 && matcher.split("?").length > 1) {
    return path.split("?")[0] === matcher.split("?")[0];
  }

  const regex = matcher.replace(/{(.*)}/, "(.*)");
  return path.match(regex) !== null;
};

const APIHistory: React.FC<Props> = ({ history, selectedRequest }) => {
  const requestHistory = history
    .sort((a, b) => b.time - a.time)
    .filter((h) => h.request && h.response)
    .filter((h) =>
      checkEquivalentPaths(selectedRequest.path ?? "", h.request.path ?? "")
    )
    .filter((h) => h.request.method === selectedRequest.method);

  if (!requestHistory.length) {
    return <p>There is no history.</p>;
  }

  return (
    <div className="flex flex-col gap-2 overflow-y-scroll max-h-[40rem]">
      {requestHistory.map((h, idx) => (
        <ApiHistoryAccordion key={idx} {...h} />
      ))}
    </div>
  );
};

function isJSON(data: string | undefined) {
  if (!data) return false;
  try {
    JSON.parse(data);
  } catch (e) {
    return false;
  }
  return true;
}

const ApiHistoryAccordion: React.FC<ApiHistoryItem> = ({
  api,
  success,
  time,
  request,
  response,
}) => {
  const [tabIndex, setTabIndex] = useState(0);

  const isJson = isJSON(atob(request.body?.toString() ?? ""));

  const tabs = [{ name: "Headers" }, { name: "Response" }];

  const jsonTabs = [...tabs, { name: "Payload" }];

  return (
    <Disclosure>
      {({ open }) => (
        <>
          <Disclosure.Button className="flex w-full justify-between rounded-lg bg-white border border-slate-100 px-4 py-2 text-left text-sm font-medium text-black hover:bg-blue-100 focus:outline-none focus-visible:ring focus-visible:ring-blue-500 focus-visible:ring-opacity-75">
            <div className="flex flex-row justify-between w-full">
              <div className="flex flex-row gap-4">
                {response.status && (
                  <Badge
                    status={success ? "green" : "red"}
                    className="!text-md"
                  >
                    Status: {response.status}
                  </Badge>
                )}
                <p className="truncate">
                  {api}
                  {request.path}
                </p>
              </div>
              <div className="flex flex-row gap-4">
                <p>{getDateString(time)}</p>
                <ChevronUpIcon
                  className={`${
                    open ? "rotate-180 transform" : ""
                  } h-5 w-5 text-blue-500`}
                />
              </div>
            </div>
          </Disclosure.Button>
          <Disclosure.Panel className="pb-2 text-sm text-gray-500">
            <div className="flex flex-col py-4">
              <div className="bg-white shadow sm:rounded-lg">
                <Tabs
                  tabs={isJson ? jsonTabs : tabs}
                  index={tabIndex}
                  setIndex={setTabIndex}
                />
                <div className="py-5">
                  {tabIndex === 0 && (
                    <TableGroup
                      headers={["Key", "Value"]}
                      rowDataClassName="max-w-[100px]"
                      groups={[
                        {
                          name: "Request Headers",
                          rows: Object.entries(request.headers)
                            .filter(([key, value]) => key && value)
                            .map(([key, value]) => [
                              key.toLowerCase(),
                              value.join(", "),
                            ]),
                        },
                        {
                          name: "Response Headers",
                          rows: Object.entries(response.headers ?? [])
                            .filter(([key, value]) => key && value)
                            .map(([key, value]) => [key.toLowerCase(), value]),
                        },
                      ]}
                    />
                  )}
                  {tabIndex === 1 && (
                    <div className="flex flex-col gap-8">
                      <div className="flex flex-col gap-2">
                        <p className="text-md font-semibold">Response Data</p>
                        <APIResponseContent
                          response={{ ...response, data: atob(response.data) }}
                        />
                      </div>
                    </div>
                  )}
                  {tabIndex === 2 && (
                    <div className="flex flex-col gap-8">
                      <div className="flex flex-col gap-2">
                        <p className="text-md font-semibold">Request Body</p>
                        <CodeEditor
                          contentType="application/json"
                          readOnly={true}
                          value={formatJSON(
                            JSON.parse(atob(request.body?.toString() ?? ""))
                          )}
                          title="Request Body"
                        />
                      </div>
                      <div className="flex flex-col gap-2">
                        {request.queryParams && (
                          <TableGroup
                            headers={["Key", "Value"]}
                            rowDataClassName="max-w-[100px]"
                            groups={[
                              {
                                name: "Query Params",
                                rows: request.queryParams
                                  .filter(({ key, value }) => key && value)
                                  .map(({ key, value }) => [key, value]),
                              },
                            ]}
                          />
                        )}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </Disclosure.Panel>
        </>
      )}
    </Disclosure>
  );
};

export default APIHistory;
