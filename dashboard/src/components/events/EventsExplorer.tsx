import { useEffect, useState } from "react";
import { useWebSocket } from "../../lib/hooks/use-web-socket";
import type {
  APIResponse,
  EventHistoryItem,
  WorkerResource,
} from "../../types";
import { Badge, Select, Spinner, Tabs, Loading } from "../shared";
import APIResponseContent from "../apis/APIResponseContent";
import {
  fieldRowArrToHeaders,
  getHost,
  generateResponse,
  formatFileSize,
  formatResponseTime,
  capitalize,
} from "../../lib/utils";
import EventsHistory from "./EventsHistory";
import { useHistory } from "../../lib/hooks/use-history";

interface Props {
  workerType: "schedules" | "topics";
}

const EventsExplorer: React.FC<Props> = ({ workerType }) => {
  const storageKey = `nitric-local-dash-${workerType}-history`;

  const { data, loading } = useWebSocket();
  const [callLoading, setCallLoading] = useState(false);

  const { data: history } = useHistory<EventHistoryItem>(workerType);

  const [response, setResponse] = useState<APIResponse>();

  const [selectedWorker, setSelectedWorker] = useState<WorkerResource>();
  const [responseTabIndex, setResponseTabIndex] = useState(0);

  const [eventHistory, setEventHistory] = useState<EventHistoryItem[]>([]);

  const handleSetCurrentWorker = (worker: WorkerResource) => {
    setSelectedWorker(worker);
  };

  const handleAppendHistory = (event: WorkerResource, success: boolean) => {
    const appendedEventHistory = [
      ...eventHistory,
      { event, success, time: Date.now() } as EventHistoryItem,
    ];

    setEventHistory(appendedEventHistory);

    localStorage.setItem(
      `${storageKey}-requests`,
      JSON.stringify(appendedEventHistory)
    );
  };

  useEffect(() => {
    if (history) {
      setEventHistory(history);
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
          (previousId &&
            data[workerType].find((s) => s.topicKey === previousId)) ||
          data[workerType][0];

        setSelectedWorker(worker);
      } else {
        // could be a refresh from ws, so update the selected endpoint
        const latest = data[workerType].find(
          (s) => s.topicKey === selectedWorker.topicKey
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
        selectedWorker.topicKey
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
      `http://${getHost()}/api/call` + `/topic/${selectedWorker.topicKey}`;
    const requestOptions: RequestInit = {
      method: "POST",
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
    handleAppendHistory(selectedWorker, callResponse.status < 400);
    setResponse(callResponse);

    setTimeout(() => setCallLoading(false), 300);
  };

  return (
    <Loading delay={400} conditionToShow={!loading}>
      {selectedWorker && data ? (
        <div className="flex max-w-7xl flex-col xl:flex-row gap-8 md:pr-8">
          <div className="w-full xl:w-7/12 flex flex-col gap-8">
            <h2 className="text-2xl font-medium text-blue-800">
              {capitalize(workerType)} - {selectedWorker?.topicKey}
            </h2>
            <div>
              <nav className="flex items-end gap-4" aria-label="Breadcrumb">
                <ol className="flex w-11/12 items-center gap-4">
                  <li className="w-full">
                    {data[workerType] && (
                      <Select<WorkerResource>
                        id={`${workerType}-select`}
                        items={data[workerType]}
                        label={capitalize(workerType)}
                        selected={selectedWorker}
                        setSelected={setSelectedWorker}
                        display={(w: WorkerResource) => (
                          <div className="flex items-center p-0.5 text-lg gap-4">
                            {w?.workerKey}
                          </div>
                        )}
                      />
                    )}
                  </li>
                </ol>
                <div className="ml-auto">
                  <button
                    type="button"
                    data-testid={`trigger-${workerType}-btn`}
                    onClick={handleSend}
                    className="inline-flex items-center rounded-md bg-blue-600 px-4 py-3 text-lg font-semibold text-white shadow-sm hover:bg-blue-500 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600"
                  >
                    Trigger
                  </button>
                </div>
              </nav>
            </div>

            <div className="bg-white shadow sm:rounded-lg">
              <div className="px-4 py-5 sm:p-6">
                <div className="sm:flex sm:items-start sm:justify-between">
                  <div className="w-full">
                    <div className="relative flex w-full">
                      <p className="text-gray-500 text-sm">
                        To initiate a POST request to{" "}
                        <a
                          data-testid="generated-request-path"
                          href={`http://${data.triggerAddress}/topic/${selectedWorker?.topicKey}`}
                          target="_blank"
                          rel="noreferrer"
                        >
                          http://{data.triggerAddress}/topic/
                          {selectedWorker?.topicKey}
                        </a>
                        , <strong>click the trigger button.</strong>
                      </p>
                    </div>
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
                                    {Object.entries(response.headers || {}).map(
                                      ([key, value]) => (
                                        <tr key={key}>
                                          <td className="whitespace-nowrap py-4 pl-4 pr-3 text-sm font-medium text-gray-900 sm:pl-6 lg:pl-8">
                                            {key}
                                          </td>
                                          <td className="whitespace-nowrap px-3 py-4 text-sm text-gray-500">
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
          <div className="w-full xl:w-5/12 flex flex-col gap-12 px-8">
            <h3 className="text-2xl font-semibold opacity-70 leading-6 text-gray-900">
              History
            </h3>
            <EventsHistory
              history={eventHistory}
              setSelectedWorker={handleSetCurrentWorker}
            />
          </div>
        </div>
      ) : !data || !data[workerType] ? (
        <div>
          Please refer to our documentation on{" "}
          <a
            className="underline"
            target="_blank"
            href="https://nitric.io/docs/"
            rel="noreferrer"
          >
            creating {workerType}
          </a>{" "}
          as we are unable to find any existing {workerType}.
        </div>
      ) : null}
    </Loading>
  );
};

export default EventsExplorer;
