import { useCallback, useEffect, useMemo, useState } from "react";
import type {
  APIRequest,
  APIResponse,
  Endpoint,
  LocalStorageHistoryItem,
} from "../../types";
import {
  Select,
  Badge,
  Spinner,
  Tabs,
  FieldRows,
  type FieldRow,
  Loading,
} from "../shared";
import {
  flattenPaths,
  generatePath,
  generatePathParams,
  formatResponseTime,
  formatFileSize,
  fieldRowArrToHeaders,
  getHost,
  generateResponse,
} from "../../lib/utils";
import APIResponseContent from "./APIResponseContent";
import CodeEditor from "./CodeEditor";
import APIMenu from "./APIMenu";
import APIHistory from "./APIHistory";

import FileUpload from "../storage/FileUpload";

import { useWebSocket } from "../../lib/hooks/use-web-socket";
import { useHistory } from "../../lib/hooks/use-history";
import AppLayout from "../layout/AppLayout";
import APITreeView from "./APITreeView";
import { copyToClipboard } from "../../lib/utils/copy-to-clipboard";
import toast from "react-hot-toast";
import { ClipboardIcon } from "@heroicons/react/24/outline";
import { APIMethodBadge } from "./APIMethodBadge";
import { Button } from "../ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";

const getTabCount = (rows: FieldRow[]) => {
  if (!rows) return 0;

  return rows.filter((r) => !!r.key).length;
};

export const LOCAL_STORAGE_KEY = "nitric-local-dash-api-history";

const requestDefault = {
  pathParams: [],
  queryParams: [
    {
      key: "",
      value: "",
    },
  ],
  headers: [
    {
      key: "Accept",
      value: "*/*",
    },
    {
      key: "User-Agent",
      value: "Nitric Client (https://www.nitric.io)",
    },
  ],
};

const bodyTabs = [
  {
    name: "JSON",
  },
  { name: "Binary" },
];

const APIExplorer = () => {
  const { data, loading } = useWebSocket();
  const [callLoading, setCallLoading] = useState(false);

  const { data: history } = useHistory("apis");

  const [JSONBody, setJSONBody] = useState<string>("");
  const [fileToUpload, setFileToUpload] = useState<File>();

  const [request, setRequest] = useState<APIRequest>(requestDefault);
  const [response, setResponse] = useState<APIResponse>();

  const [selectedApiEndpoint, setSelectedApiEndpoint] = useState<Endpoint>();
  const [currentTabIndex, setCurrentTabIndex] = useState(0);
  const [bodyTabIndex, setBodyTabIndex] = useState(0);
  const [responseTabIndex, setResponseTabIndex] = useState(0);

  const [requiredPathParamErrors, setRequiredPathParamErrors] = useState({});

  const paths = useMemo(
    () => data?.apis?.map((doc) => flattenPaths(doc)).flat(),
    [data]
  );

  // Load single history from localStorage on mount
  useEffect(() => {
    if (selectedApiEndpoint) {
      const storedHistory = localStorage.getItem(
        `${LOCAL_STORAGE_KEY}-${selectedApiEndpoint.id}`
      );

      if (storedHistory) {
        const history: LocalStorageHistoryItem = JSON.parse(storedHistory);
        setJSONBody(history.JSONBody);
        setRequest((prev) => ({
          ...prev,
          ...history.request,
          pathParams: generatePathParams(selectedApiEndpoint, history.request),
        }));
      } else {
        // clear
        setJSONBody("");
        setRequest((prev) => ({
          ...prev,
          ...requestDefault,
          pathParams: generatePathParams(selectedApiEndpoint, requestDefault),
        }));
      }

      // set history
      localStorage.setItem(
        `${LOCAL_STORAGE_KEY}-last-path-id`,
        selectedApiEndpoint.id
      );

      // clear response
      setResponse(undefined);
    }
  }, [selectedApiEndpoint]);

  // Load request history
  useEffect(() => {
    const localHistory = localStorage.getItem(`${LOCAL_STORAGE_KEY}-requests`);
    if (!localHistory) {
      localStorage.setItem(`${LOCAL_STORAGE_KEY}-requests`, JSON.stringify([]));
      return;
    }
  }, []);

  useEffect(() => {
    if (paths?.length) {
      // restore history or select first if not selected
      if (!selectedApiEndpoint) {
        const previousId = localStorage.getItem(
          `${LOCAL_STORAGE_KEY}-last-path-id`
        );

        const path =
          (previousId && paths.find((p) => p.id === previousId)) || paths[0];

        setSelectedApiEndpoint(path);
        setRequest((prev) => ({
          ...prev,
          method: path.method,
        }));
      } else {
        // could be a refresh from ws, so update the selected endpoint
        const latest = paths.find((p) => p.id === selectedApiEndpoint.id);

        if (latest) {
          setSelectedApiEndpoint(latest);
          setRequest((prev) => ({
            ...prev,
            method: latest.method,
          }));
        }
      }
    }
  }, [paths]);

  useEffect(() => {
    if (selectedApiEndpoint) {
      const generatedPath = generatePath(
        selectedApiEndpoint.path,
        request.pathParams,
        request.queryParams
      );

      setRequest((prev) => ({
        ...prev,
        path: generatedPath,
        method: selectedApiEndpoint.method,
      }));
    }
  }, [selectedApiEndpoint, request.pathParams, request.queryParams]);

  // Save state to local storage whenever it changes
  useEffect(() => {
    if (selectedApiEndpoint) {
      localStorage.setItem(
        `${LOCAL_STORAGE_KEY}-${selectedApiEndpoint.id}`,
        JSON.stringify({
          request,
          JSONBody,
        })
      );
    }
  }, [request, JSONBody]);

  const onDrop = useCallback(
    async (acceptedFiles: File[]) => setFileToUpload(acceptedFiles[0]),
    []
  );

  const apiAddress =
    selectedApiEndpoint && data?.apiAddresses
      ? data.apiAddresses[selectedApiEndpoint.api]
      : null;

  const tabs = [
    {
      name: "Params",
      count: getTabCount(request.queryParams) + getTabCount(request.pathParams),
    },
    { name: "Headers", count: getTabCount(request.headers) },
    { name: "Body", count: JSONBody ? 1 : undefined },
  ];

  const currentTabName = tabs[currentTabIndex].name;

  const currentBodyTabName = bodyTabs[bodyTabIndex].name;

  const refreshPathParamErrors = () => {
    const newPathParamErrors: Record<number, FieldRow> = {};
    const emptyParams = request.pathParams.filter((p, idx) => {
      if (p.value === "") {
        newPathParamErrors[idx] = p;
        return true;
      }

      return false;
    });

    setRequiredPathParamErrors(newPathParamErrors);

    return emptyParams;
  };

  useEffect(() => {
    if (Object.keys(requiredPathParamErrors).length) {
      refreshPathParamErrors();
    }
  }, [request.pathParams]);

  const handleSend = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => {
    if (!selectedApiEndpoint) return;
    setCallLoading(true);
    e.preventDefault();

    if (request.pathParams.length) {
      const emptyParams = refreshPathParamErrors();

      if (emptyParams.length) {
        setCallLoading(false);
        toast.error(
          `Required path parameter(s) missing: ${emptyParams
            .map((p) => p.key)
            .join(", ")}`
        );

        return;
      }
    }

    const { path, method, headers } = request;

    const url = `http://${getHost()}/api/call` + path;
    const requestOptions: RequestInit = {
      method,
      headers: fieldRowArrToHeaders([
        ...headers,
        {
          key: "X-Nitric-Local-Call-Address",
          value: apiAddress || "localhost:4001",
        },
      ]),
    };

    if (method !== "GET" && method !== "HEAD") {
      // handle body in request
      if (currentBodyTabName === "Binary" && fileToUpload) {
        requestOptions.body = fileToUpload;
      } else if (currentBodyTabName === "JSON" && JSONBody.trim()) {
        requestOptions.body = JSONBody;
      }
    }
    const startTime = window.performance.now();
    const res = await fetch(url, requestOptions);

    const callResponse = await generateResponse(res, startTime);
    setResponse(callResponse);

    setTimeout(() => setCallLoading(false), 300);
  };

  return (
    <AppLayout
      title="API Explorer"
      routePath={"/"}
      secondLevelNav={
        paths && selectedApiEndpoint && request?.method ? (
          <>
            <div className="flex mb-2 items-center justify-between px-2">
              <span className="text-lg">APIs</span>
              <APIMenu
                selected={selectedApiEndpoint}
                onAfterClear={() => {
                  setJSONBody("");
                  setRequest({
                    ...requestDefault,
                    method: selectedApiEndpoint.method,
                    path: generatePath(selectedApiEndpoint.path, [], []),
                    pathParams: generatePathParams(
                      selectedApiEndpoint,
                      requestDefault
                    ),
                  });
                }}
              />
            </div>
            <APITreeView
              defaultTreeIndex={selectedApiEndpoint.id}
              onSelect={(endpoint) => {
                setSelectedApiEndpoint(endpoint);
              }}
              endpoints={paths}
            />
          </>
        ) : null
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {paths && selectedApiEndpoint && request?.method ? (
          <div className="flex max-w-6xl flex-col gap-8 md:pr-8">
            <div className="w-full flex flex-col gap-8">
              <div>
                <div className="flex md:hidden">
                  <h2 className="text-2xl">API - {selectedApiEndpoint.api}</h2>
                  <APIMenu
                    selected={selectedApiEndpoint}
                    onAfterClear={() => {
                      setJSONBody("");
                      setRequest({
                        ...requestDefault,
                        method: selectedApiEndpoint.method,
                        path: generatePath(selectedApiEndpoint.path, [], []),
                        pathParams: generatePathParams(
                          selectedApiEndpoint,
                          requestDefault
                        ),
                      });
                    }}
                  />
                </div>
                <nav
                  className="flex items-end lg:items-center gap-4"
                  aria-label="Breadcrumb"
                >
                  <ol className="flex w-full items-center lg:hidden gap-4">
                    <li className="w-full">
                      <Select<Endpoint>
                        items={paths}
                        label="Endpoint"
                        id="endpoint-select"
                        selected={selectedApiEndpoint}
                        setSelected={setSelectedApiEndpoint}
                        display={(v) => (
                          <div className="grid grid-cols-12 items-center p-0.5 text-lg gap-4">
                            <APIMethodBadge
                              method={v.method}
                              className="!text-lg col-span-3 md:col-span-2"
                            />
                            <div className="col-span-9 md:col-span-10 flex gap-4">
                              <span>{v?.api}</span>
                              <span>{v?.path}</span>
                            </div>
                          </div>
                        )}
                      />
                    </li>
                  </ol>
                  <div className="hidden lg:flex items-center gap-4">
                    <APIMethodBadge
                      className="!text-lg"
                      method={request.method}
                    />
                    <span
                      className="text-lg flex gap-2"
                      data-testid="generated-request-path"
                    >
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="truncate max-w-xl">
                            {request.path}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>{request.path}</p>
                        </TooltipContent>
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button
                            type="button"
                            onClick={() => {
                              copyToClipboard(
                                `http://${apiAddress}${request.path}`
                              );
                              toast.success("Copied Route URL");
                            }}
                          >
                            <span className="sr-only">Copy Route URL</span>
                            <ClipboardIcon className="w-5 h-5 text-gray-500" />
                          </button>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>Copy Route URL</p>
                        </TooltipContent>
                      </Tooltip>
                    </span>
                  </div>
                  <div className="ml-auto">
                    <Button
                      size="lg"
                      data-testid="send-api-btn"
                      onClick={handleSend}
                    >
                      Send
                    </Button>
                  </div>
                </nav>
              </div>

              <div className="bg-white shadow sm:rounded-lg">
                <Tabs
                  tabs={tabs}
                  index={currentTabIndex}
                  setIndex={setCurrentTabIndex}
                />
                <div className="px-4 py-5 sm:p-6">
                  <div className="sm:flex sm:items-start sm:justify-between">
                    <div className="w-full">
                      <div className="relative flex w-full">
                        <h3 className="text-xl font-semibold leading-6 text-gray-900">
                          {currentTabName}
                        </h3>
                      </div>
                      {currentTabName === "Params" && (
                        <ul className="divide-gray-200 my-4">
                          {request.pathParams.length > 0 && (
                            <li className="flex flex-col py-4">
                              <h4 className="text-lg font-medium text-gray-900">
                                Path Params
                              </h4>
                              <FieldRows
                                lockKeys
                                testId="path"
                                valueRequired
                                rows={request.pathParams}
                                valueErrors={requiredPathParamErrors}
                                setRows={(rows) => {
                                  setRequest((prev) => ({
                                    ...prev,
                                    pathParams: rows,
                                  }));
                                }}
                              />
                            </li>
                          )}
                          <li className="flex flex-col py-4">
                            <h4 className="text-lg font-medium text-gray-900">
                              Query Params
                            </h4>
                            <FieldRows
                              rows={request.queryParams}
                              testId="query"
                              setRows={(rows) => {
                                setRequest((prev) => ({
                                  ...prev,
                                  queryParams: rows,
                                }));
                              }}
                            />
                          </li>
                        </ul>
                      )}
                      {currentTabName === "Headers" && (
                        <div className="my-4">
                          <FieldRows
                            rows={request.headers}
                            testId="header"
                            setRows={(rows) => {
                              setRequest((prev) => ({
                                ...prev,
                                headers: rows,
                              }));
                            }}
                          />
                        </div>
                      )}
                      {currentTabName === "Body" && (
                        <div className="my-4 flex flex-col gap-4">
                          <Tabs
                            tabs={bodyTabs}
                            index={bodyTabIndex}
                            pill
                            setIndex={setBodyTabIndex}
                          />
                          {currentBodyTabName === "JSON" && (
                            <CodeEditor
                              id="json-editor"
                              contentType={"application/json"}
                              value={JSONBody}
                              includeLinters
                              onChange={(value) => {
                                setJSONBody(value);
                              }}
                            />
                          )}
                          {currentBodyTabName === "Binary" && (
                            <div className="flex flex-col mb-2">
                              <h4 className="text-lg mb-2 font-medium text-gray-900">
                                Binary File
                              </h4>
                              <FileUpload multiple={false} onDrop={onDrop} />
                              {fileToUpload && (
                                <span
                                  data-testid="file-upload-info"
                                  className="px-4 flex items-center py-4 sm:px-0"
                                >
                                  {fileToUpload.name} -{" "}
                                  {formatFileSize(fileToUpload.size)}
                                </span>
                              )}
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
              <div className="bg-white shadow sm:rounded-lg">
                <div className="px-4 py-5 sm:p-6">
                  <div className="sm:flex sm:items-start sm:justify-between">
                    <div className="w-full relative">
                      <div className="flex items-center gap-4">
                        <h3 className="text-xl font-semibold leading-6 text-gray-900">
                          Response
                        </h3>
                        {callLoading && (
                          <Spinner
                            className="absolute top-0"
                            color="info"
                            size={"md"}
                          />
                        )}
                      </div>
                      <div className="absolute right-0 top-0 flex gap-2">
                        {response?.status && (
                          <Badge
                            data-testid="response-status"
                            status={response.status >= 400 ? "red" : "green"}
                          >
                            Status: {response.status}
                          </Badge>
                        )}
                        {response?.time && (
                          <Badge data-testid="response-time" status={"green"}>
                            Time: {formatResponseTime(response.time)}
                          </Badge>
                        )}
                        {typeof response?.size === "number" && (
                          <Badge data-testid="response-size" status={"green"}>
                            Size: {formatFileSize(response.size)}
                          </Badge>
                        )}
                      </div>

                      <div className="my-4 max-w-full text-sm">
                        {response?.data ? (
                          <div className="flex flex-col gap-4">
                            <Tabs
                              tabs={[
                                {
                                  name: "Response",
                                },
                                {
                                  name: "Headers",
                                  count: Object.keys(response.headers || {})
                                    .length,
                                },
                              ]}
                              round
                              index={responseTabIndex}
                              setIndex={setResponseTabIndex}
                            />
                            {responseTabIndex === 0 && (
                              <APIResponseContent response={response} />
                            )}
                            {responseTabIndex === 1 && (
                              <div className="overflow-x-auto">
                                <div className="inline-block min-w-full py-2 align-middle">
                                  <table className="min-w-full divide-y divide-gray-300">
                                    <thead>
                                      <tr>
                                        <th
                                          scope="col"
                                          className="py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6 lg:pl-8"
                                        >
                                          Header
                                        </th>
                                        <th
                                          scope="col"
                                          className="px-3 py-3.5 text-left text-sm font-semibold text-gray-900"
                                        >
                                          Value
                                        </th>
                                      </tr>
                                    </thead>
                                    <tbody className="divide-y divide-gray-200 bg-white">
                                      {Object.entries(
                                        response.headers || {}
                                      ).map(([key, value]) => (
                                        <tr key={key}>
                                          <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6 lg:pl-8">
                                            {key}
                                          </td>
                                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
                                            {value}
                                          </td>
                                        </tr>
                                      ))}
                                    </tbody>
                                  </table>
                                </div>
                              </div>
                            )}
                          </div>
                        ) : response ? (
                          <span className="text-gray-500 text-lg">
                            No response data available for this request.
                          </span>
                        ) : (
                          <span className="text-gray-500 text-lg">
                            Send a request to get a response.
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div className="w-full flex flex-col gap-8 pb-20">
              <h3 className="text-2xl font-semibold leading-6">
                Request History
              </h3>
              <APIHistory
                history={history?.apis ?? []}
                selectedRequest={{
                  path: selectedApiEndpoint.path,
                  method: request.method,
                }}
              />
            </div>
          </div>
        ) : (
          <div>
            Please refer to our documentation on{" "}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/apis"
              rel="noreferrer"
            >
              creating APIs
            </a>{" "}
            as we are unable to find any existing APIs.
          </div>
        )}
      </Loading>
    </AppLayout>
  );
};

export default APIExplorer;
