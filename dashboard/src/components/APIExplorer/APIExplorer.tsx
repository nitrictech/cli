import { useEffect, useMemo, useState, lazy, Suspense } from "react";
import { useWebSocket } from "../../lib/use-web-socket";
import Select from "../shared/Select";
import type {
  APIRequest,
  APIResponse,
  Endpoint,
  HistoryItem,
  Method,
} from "../../types";
import { CodeBlock } from "../shared/CodeBlock";
import Badge from "../shared/Badge";
import {
  fieldRowArrToHeaders,
  getHost,
  headersToObject,
} from "../../lib/utils";
import Spinner from "../shared/Spinner";
import Tabs from "../layout/Tabs";
import FieldRows, { FieldRow } from "../shared/FieldRows";
import type { JSONEditorProps } from "../shared/JSONEditor";
import { flattenPaths } from "./flatten-paths";
import { generatePath } from "./generate-path";
const JSONEditor = lazy(() => import("../shared/JSONEditor")); // Lazy-loaded

const getTabCount = (rows: FieldRow[]) => rows.filter((r) => !!r.key).length;

// const LOCAL_STORAGE_KEY = "nitric-local-dash";

// const MAX_HISTORY_LENGTH = 50;

const APIExplorer = () => {
  //const [history, setHistory] = useState<HistoryItem[]>([]);
  const { data } = useWebSocket();
  const [callLoading, setCallLoading] = useState(false);

  const [JSONBody, setJSONBody] = useState<JSONEditorProps["content"]>({
    text: "",
  });

  const [request, setRequest] = useState<APIRequest>({
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
  });
  const [response, setResponse] = useState<APIResponse>();

  const [selectedApiEndpoint, setSelectedApiEndpoint] = useState<Endpoint>();
  const [currentTabIndex, setCurrentTabIndex] = useState(0);
  const [responseTabIndex, setResponseTabIndex] = useState(0);

  const paths = useMemo(
    () => data?.apis.map((doc) => flattenPaths(doc)).flat(),
    [data]
  );

  // Load history from localStorage on mount
  // useEffect(() => {
  //   const storedHistory = localStorage.getItem(
  //     `${LOCAL_STORAGE_KEY}-call-history`
  //   );
  //   if (storedHistory) {
  //     setHistory(JSON.parse(storedHistory));
  //   }
  // }, []);

  useEffect(() => {
    if (paths?.length) {
      setSelectedApiEndpoint(paths[0]);
      setRequest((prev) => ({
        ...prev,
        method: paths[0].methods[0],
      }));
    }
  }, [paths]);

  useEffect(() => {
    if (request.method && selectedApiEndpoint) {
      const propsToMerge: Record<string, any> = {};
      // Save state to local storage
      //console.log("saving", selectedApiEndpoint);

      if (!selectedApiEndpoint.methods.includes(request.method)) {
        propsToMerge.method = selectedApiEndpoint.methods[0];
      }

      if (selectedApiEndpoint.params?.length) {
        const pathParams: FieldRow[] = [];

        selectedApiEndpoint.params.forEach((p) => {
          p.value.forEach((v) => {
            if (v.in === "path") {
              pathParams.push({
                key: v.name,
                value: "",
              });
            }
          });
        });

        propsToMerge.pathParams = pathParams;
      }

      console.count("hi");
      console.log(propsToMerge, Object.keys(propsToMerge).length);

      if (Object.keys(propsToMerge).length) {
        setRequest((prev) => ({
          ...prev,
          ...propsToMerge,
        }));
      }
    }
  }, [selectedApiEndpoint, request.method]);

  useEffect(() => {
    if (selectedApiEndpoint) {
      const generatedPath = generatePath(
        selectedApiEndpoint,
        request.pathParams,
        request.queryParams
      );

      setRequest((prev) => ({
        ...prev,
        path: generatedPath,
      }));
    }
  }, [selectedApiEndpoint, request.pathParams, request.queryParams]);

  console.log("request", request);

  // Add item to history and persist to localStorage
  // const addToHistory = (item: HistoryItem) => {
  //   const updatedHistory = [...history, item].slice(-MAX_HISTORY_LENGTH);
  //   setHistory(updatedHistory);
  //   localStorage.setItem(
  //     `${LOCAL_STORAGE_KEY}-call-history`,
  //     JSON.stringify(updatedHistory)
  //   );
  // };

  if (!paths || !selectedApiEndpoint) {
    return null;
  }

  const handleSend = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => {
    if (!selectedApiEndpoint) return;
    setCallLoading(true);
    e.preventDefault();

    const { path, method, headers } = request;

    const url = `http://${getHost()}/call` + path;
    const requestOptions: RequestInit = {
      method,
      headers: fieldRowArrToHeaders(headers),
    };

    const jsonBody = (JSONBody as { text: string }).text;

    if (method !== "GET" && method !== "HEAD" && jsonBody) {
      requestOptions.body = JSON.stringify(
        JSONBody ? JSON.parse(jsonBody) : {}
      );
    }
    const startTime = window.performance.now();
    const res = await fetch(url, requestOptions);

    const data =
      res.headers.get("Content-Type") === "application/json"
        ? await res.json()
        : await res.text();

    const endTime = window.performance.now();
    const responseSize = res.headers.get("Content-Length");

    const callResponse = {
      data: JSON.stringify(data, null, 2),
      time: endTime - startTime,
      status: res.status,
      size: responseSize ? parseInt(responseSize) : 0,
      headers: headersToObject(res.headers),
    };

    setResponse(callResponse);
    //   addToHistory({
    //     request: {
    //       ...request,
    //       body: requestOptions.body,
    //     },
    //     response: callResponse,
    //     time: new Date().getTime(),
    //   });
    setTimeout(() => setCallLoading(false), 300);
  };

  //console.log("request", request);

  //console.log("response", response);

  const tabs = [
    {
      name: "Params",
      count: getTabCount(request.queryParams) + getTabCount(request.pathParams),
    },
    { name: "Headers", count: getTabCount(request.headers) },
    { name: "Body", count: JSONBody ? 1 : undefined },
  ];

  const currentTabName = tabs[currentTabIndex].name;

  return (
    <div className='flex flex-col md:flex-row gap-8 pr-8'>
      <div className='md:w-1/2 flex flex-col gap-8'>
        <nav className='flex items-end gap-4' aria-label='Breadcrumb'>
          <ol role='list' className='flex w-11/12 items-center gap-4'>
            <li className='w-9/12'>
              <Select<Endpoint>
                items={paths}
                label='API Endpoint'
                selected={selectedApiEndpoint}
                setSelected={setSelectedApiEndpoint}
                display={(v) => (
                  <div className='flex items-center p-0.5 text-lg gap-4'>
                    <span>{v?.api}</span>
                    <span>{v?.path}</span>
                    <span className='ml-auto px-2 text-sm'>
                      {v?.methods.length} methods
                    </span>
                  </div>
                )}
              />
            </li>
            <li className='w-3/12'>
              <Select<Method>
                items={selectedApiEndpoint?.methods || []}
                label='Method'
                selected={request.method}
                setSelected={(m) => {
                  setRequest((prev) => ({
                    ...prev,
                    method: m,
                  }));
                }}
                display={(method) => (
                  <Badge
                    status={
                      (
                        {
                          DELETE: "red",
                          POST: "green",
                          PUT: "yellow",
                          GET: "blue",
                        } as any
                      )[method]
                    }
                    className='!text-lg'
                  >
                    {method}
                  </Badge>
                )}
              />
            </li>
          </ol>
          <div className='ml-auto'>
            <button
              type='button'
              onClick={handleSend}
              className='inline-flex items-center rounded-md bg-blue-600 px-4 py-3 text-lg font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600'
            >
              Send
            </button>
          </div>
        </nav>
        <div className='bg-white shadow sm:rounded-lg'>
          <Tabs
            tabs={tabs}
            index={currentTabIndex}
            setIndex={setCurrentTabIndex}
          />
          <div className='px-4 py-5 sm:p-6'>
            <div className='sm:flex sm:items-start sm:justify-between'>
              <div className='w-full'>
                <div className='relative flex w-full'>
                  <h3 className='text-xl font-semibold leading-6 text-gray-900'>
                    {currentTabName}
                  </h3>
                  <p className='absolute text-gray-500 text-sm top-0 right-0'>
                    <a
                      href={`http://localhost:4001${request.path}`}
                      target='_blank'
                    >
                      http://localhost:4001
                      {request.path}
                    </a>
                  </p>
                </div>
                {currentTabName === "Params" && (
                  <ul role='list' className='divide-gray-200 my-4'>
                    <li className='flex flex-col py-4'>
                      <h4 className='text-lg font-medium text-gray-900'>
                        Query Params
                      </h4>
                      <FieldRows
                        rows={request.queryParams}
                        setRows={(rows) => {
                          setRequest((prev) => ({
                            ...prev,
                            queryParams: rows,
                          }));
                        }}
                      />
                    </li>
                    {request.pathParams.length > 0 && (
                      <li className='flex flex-col py-4'>
                        <h4 className='text-lg font-medium text-gray-900'>
                          Path Params
                        </h4>
                        <FieldRows
                          lockKeys
                          rows={request.pathParams}
                          setRows={(rows) => {
                            setRequest((prev) => ({
                              ...prev,
                              pathParams: rows,
                            }));
                          }}
                        />
                      </li>
                    )}
                  </ul>
                )}
                {currentTabName === "Headers" && (
                  <div className='my-4'>
                    <FieldRows
                      rows={request.headers}
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
                  <div className='my-4'>
                    <Suspense>
                      <JSONEditor content={JSONBody} onChange={setJSONBody} />
                    </Suspense>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        <div className='bg-white shadow sm:rounded-lg'>
          <div className='px-4 py-5 sm:p-6'>
            <div className='sm:flex sm:items-start sm:justify-between'>
              <div className='w-full relative'>
                <div className='flex items-center gap-4'>
                  <h3 className='text-xl font-semibold leading-6 text-gray-900'>
                    Response
                  </h3>
                  {callLoading && (
                    <Spinner
                      className='absolute top-0'
                      color='info'
                      size={"md"}
                    />
                  )}
                </div>
                <div className='absolute right-0 top-0 flex gap-2'>
                  {response?.status && (
                    <Badge status={response.status >= 400 ? "red" : "green"}>
                      Status: {response.status}
                    </Badge>
                  )}
                  {response?.time && (
                    <Badge status={"green"}>Time: {response.time} ms</Badge>
                  )}
                  {response?.size && (
                    <Badge status={"green"}>Size: {response.size} bytes</Badge>
                  )}
                </div>

                <div className='my-4 max-w-full text-sm'>
                  {response?.data ? (
                    <div className='flex flex-col gap-4'>
                      <Tabs
                        tabs={[
                          {
                            name: "Response",
                          },
                          {
                            name: "Headers",
                            count: Object.keys(response.headers || {}).length,
                          },
                        ]}
                        round
                        index={responseTabIndex}
                        setIndex={setResponseTabIndex}
                      />
                      {responseTabIndex === 0 && (
                        <CodeBlock>{response?.data || ""}</CodeBlock>
                      )}
                      {responseTabIndex === 1 && (
                        <div className='overflow-x-auto'>
                          <div className='inline-block min-w-full py-2 align-middle'>
                            <table className='min-w-full divide-y divide-gray-300'>
                              <thead>
                                <tr>
                                  <th
                                    scope='col'
                                    className='py-3.5 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-6 lg:pl-8'
                                  >
                                    Header
                                  </th>
                                  <th
                                    scope='col'
                                    className='px-3 py-3.5 text-left text-sm font-semibold text-gray-900'
                                  >
                                    Value
                                  </th>
                                </tr>
                              </thead>
                              <tbody className='divide-y divide-gray-200 bg-white'>
                                {Object.entries(response.headers || {}).map(
                                  ([key, value]) => (
                                    <tr key={key}>
                                      <td className='whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6 lg:pl-8'>
                                        {key}
                                      </td>
                                      <td className='whitespace-nowrap px-3 py-4 text-sm text-gray-500'>
                                        {value}
                                      </td>
                                    </tr>
                                  )
                                )}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      )}
                    </div>
                  ) : (
                    <span className='text-gray-500 text-lg'>
                      Send a request to get a response.
                    </span>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className='w-1/2 flex flex-col gap-12 px-8'>
        <h3 className='text-2xl font-semibold leading-6 text-gray-900'>
          History
        </h3>
        {/* <APIHistory history={history} /> */}
      </div>
    </div>
  );
};

export default APIExplorer;
