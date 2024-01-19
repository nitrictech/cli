import { useEffect, useState } from "react";
import { useWebSocket } from "../../lib/hooks/use-web-socket";
import type { APIResponse, EventHistoryItem, Schedule, Topic } from "@/types";
import { Badge, Select, Spinner, Tabs, Loading } from "../shared";
import APIResponseContent from "../apis/APIResponseContent";
import {
  fieldRowArrToHeaders,
  getHost,
  generateResponse,
  formatFileSize,
  formatResponseTime,
  formatJSON,
} from "../../lib/utils";
import EventsHistory from "./EventsHistory";
import { useHistory } from "../../lib/hooks/use-history";
import CodeEditor from "../apis/CodeEditor";
import EventsMenu from "./EventsMenu";
import AppLayout from "../layout/AppLayout";
import EventsTreeView from "./EventsTreeView";
import { copyToClipboard } from "../../lib/utils/copy-to-clipboard";
import ClipboardIcon from "@heroicons/react/24/outline/ClipboardIcon";
import toast from "react-hot-toast";
import { capitalize } from "radash";
import { Button } from "../ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";

interface Props {
  workerType: "schedules" | "topics";
}

type Worker = Schedule | Topic;

const EventsExplorer: React.FC<Props> = ({ workerType }) => {
  const storageKey = `nitric-local-dash-${workerType}-history`;

  const { data, loading } = useWebSocket();
  const [callLoading, setCallLoading] = useState(false);

  const { data: history } = useHistory(workerType);

  const [response, setResponse] = useState<APIResponse>();

  const [selectedWorker, setSelectedWorker] = useState<Worker>();
  const [responseTabIndex, setResponseTabIndex] = useState(0);

  const [eventHistory, setEventHistory] = useState<EventHistoryItem[]>([]);

  const [body, setBody] = useState({});

  useEffect(() => {
    if (history) {
      setEventHistory(history ? history[workerType] : []);
    }
  }, [history]);

  useEffect(() => {
    if (data && data[workerType]) {
      // restore history or select first if not selected
      if (!selectedWorker) {
        const previousId = localStorage.getItem(
          `${storageKey}-last-${workerType}`
        );

        const worker =
          (previousId && data[workerType].find((s) => s.name === previousId)) ||
          data[workerType][0];

        setSelectedWorker(worker);
      } else {
        // could be a refresh from ws, so update the selected endpoint
        const latest = data[workerType].find(
          (s) => s.name === selectedWorker.name
        );

        if (latest) {
          setSelectedWorker(latest);
        }
      }
    }
  }, [data]);

  useEffect(() => {
    if (selectedWorker) {
      // set history
      localStorage.setItem(
        `${storageKey}-last-${workerType}`,
        selectedWorker.name
      );
    }
  }, [selectedWorker]);

  const handleSend = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => {
    if (!selectedWorker) return;
    setCallLoading(true);
    e.preventDefault();

    const url =
      `http://${getHost()}/api/call` + `/${workerType}/${selectedWorker.name}`;
    const requestOptions: RequestInit = {
      method: "POST",
      body: JSON.stringify(body),
      headers: fieldRowArrToHeaders([
        {
          key: "Accept",
          value: "*/*",
        },
        {
          key: "User-Agent",
          value: "Nitric Client (https://www.nitric.io)",
        },
        {
          key: "X-Nitric-Local-Call-Address",
          value: data?.triggerAddress || "localhost:4000",
        },
      ]),
    };

    const startTime = window.performance.now();
    const res = await fetch(url, requestOptions);

    const callResponse = await generateResponse(res, startTime);
    setResponse(callResponse);

    setTimeout(() => setCallLoading(false), 300);
  };

  const workerTitleSingle = capitalize(workerType).slice(0, -1);
  const generatedURL = `http://${data?.triggerAddress}/${workerType}/${selectedWorker?.name}`;

  const hasData = Boolean(data && data[workerType]?.length);

  return (
    <AppLayout
      title={capitalize(workerType)}
      routePath={`/${workerType}`}
      secondLevelNav={
        data &&
        selectedWorker && (
          <>
            <div className="flex mb-2 items-center justify-between px-2">
              <span className="text-lg">{capitalize(workerType)}</span>
              <EventsMenu
                selected={selectedWorker.name}
                storageKey={storageKey}
                workerType={workerType}
                onAfterClear={() => {
                  return;
                }}
              />
            </div>
            <EventsTreeView
              initialItem={selectedWorker}
              onSelect={(resource) => {
                setSelectedWorker(resource);
              }}
              resources={data[workerType] ?? []}
            />
          </>
        )
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {selectedWorker && hasData ? (
          <div className="flex max-w-6xl flex-col gap-8 md:pr-8">
            <div className="w-full flex flex-col gap-8">
              <div className="flex">
                <h2 className="text-2xl">{selectedWorker.name}</h2>
                <div className="flex ml-auto items-center md:hidden">
                  <EventsMenu
                    selected={selectedWorker.name}
                    storageKey={storageKey}
                    workerType={workerType}
                    onAfterClear={() => {
                      return;
                    }}
                  />
                </div>
              </div>
              <div>
                <nav className="flex items-end gap-4" aria-label="Breadcrumb">
                  <ol className="flex md:hidden w-11/12 items-center gap-4">
                    <li className="w-full">
                      {data[workerType] && (
                        <Select
                          id={`${workerType}-select`}
                          items={data[workerType]}
                          label={capitalize(workerType)}
                          selected={selectedWorker}
                          setSelected={setSelectedWorker}
                          display={(w: Worker) => (
                            <div className="flex items-center p-0.5 text-lg gap-4">
                              {w.name}
                            </div>
                          )}
                        />
                      )}
                    </li>
                  </ol>
                  <span className="text-lg hidden md:flex gap-2">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <span className="truncate max-w-lg">
                          {generatedURL}
                        </span>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>{generatedURL}</p>
                      </TooltipContent>
                    </Tooltip>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <button
                          type="button"
                          onClick={() => {
                            copyToClipboard(generatedURL);
                            toast.success(`Copied ${workerTitleSingle} URL`);
                          }}
                        >
                          <span className="sr-only">
                            Copy {workerTitleSingle} URL
                          </span>
                          <ClipboardIcon className="w-5 h-5 text-gray-500" />
                        </button>
                      </TooltipTrigger>
                      <TooltipContent>
                        <p>Copy {workerTitleSingle} URL</p>
                      </TooltipContent>
                    </Tooltip>
                  </span>
                  <span className="hidden md:block"></span>
                  {workerType === "schedules" && (
                    <Button
                      size="lg"
                      className="ml-auto"
                      data-testid={`trigger-${workerType}-btn`}
                      onClick={handleSend}
                    >
                      Trigger
                    </Button>
                  )}
                </nav>
              </div>

              {workerType === "topics" && (
                <div className="flex flex-col py-4">
                  <div className="bg-white shadow sm:rounded-lg px-4 py-5 sm:p-6">
                    <h2>Payload</h2>
                    <div>
                      <CodeEditor
                        value={formatJSON(body)}
                        contentType="application/json"
                        onChange={(payload: string) => {
                          try {
                            setBody(JSON.parse(payload));
                          } catch {
                            return;
                          }
                        }}
                      />

                      <Button
                        size="lg"
                        className="ml-auto flex mt-6"
                        data-testid={`trigger-${workerType}-btn`}
                        onClick={handleSend}
                      >
                        {workerType === "topics" ? "Publish" : "Trigger"}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
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
                            status={response.status >= 400 ? "red" : "green"}
                          >
                            Status: {response.status}
                          </Badge>
                        )}
                        {response?.time && (
                          <Badge status={"green"}>
                            Time: {formatResponseTime(response.time)}
                          </Badge>
                        )}
                        {typeof response?.size === "number" && (
                          <Badge status={"green"}>
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
              <h3 className="text-2xl font-semibold leading-6">History</h3>
              <EventsHistory
                history={eventHistory}
                workerType={workerType}
                selectedWorker={selectedWorker}
              />
            </div>
          </div>
        ) : !hasData ? (
          <div>
            Please refer to our documentation on{" "}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/messaging"
              rel="noreferrer"
            >
              creating {capitalize(workerType)}
            </a>{" "}
            as we are unable to find any existing {workerType}.
          </div>
        ) : null}
      </Loading>
    </AppLayout>
  );
};

export default EventsExplorer;
