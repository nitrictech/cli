import { useEffect, useState } from "react";
import { useWebSocket } from "../lib/use-web-socket";
import Select from "./shared/Select";
import type { APIResponse, Schedule } from "../types";
import Badge from "./shared/Badge";
import Spinner from "./shared/Spinner";
import { formatFileSize } from "./APIExplorer/format-file-size";
import Tabs from "./layout/Tabs";
import APIResponseContent from "./APIExplorer/APIResponseContent";
import { fieldRowArrToHeaders, getHost } from "../lib/utils";
import { generateResponse } from "../lib/generate-response";
import { formatResponseTime } from "./APIExplorer/format-response-time";

export const LOCAL_STORAGE_KEY = "nitric-local-dash-schedule-history";

const ScheduleExplorer = () => {
  const { data } = useWebSocket();
  const [callLoading, setCallLoading] = useState(false);

  const [response, setResponse] = useState<APIResponse>();

  const [selectedSchedule, setSelectedSchedule] = useState<Schedule>();
  const [responseTabIndex, setResponseTabIndex] = useState(0);

  useEffect(() => {
    if (data?.schedules.length) {
      // restore history or select first if not selected
      if (!selectedSchedule) {
        const previousId = localStorage.getItem(
          `${LOCAL_STORAGE_KEY}-last-schedule`
        );

        const schedule =
          (previousId &&
            data.schedules.find((s) => s.topicKey === previousId)) ||
          data.schedules[0];

        setSelectedSchedule(schedule);
      } else {
        // could be a refresh from ws, so update the selected endpoint
        const latest = data.schedules.find(
          (s) => s.topicKey === selectedSchedule.topicKey
        );

        if (latest) {
          setSelectedSchedule(latest);
        }
      }
    }
  }, [data]);

  if (!data?.schedules.length) {
    return (
      <div>
        Please refer to our documentation on{" "}
        <a
          className="underline"
          target="_blank"
          href="https://nitric.io/docs/schedules#create-schedules"
        >
          creating schedules
        </a>{" "}
        as we are unable to find any existing schedules.
      </div>
    );
  }

  const handleSend = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => {
    if (!selectedSchedule) return;
    setCallLoading(true);
    e.preventDefault();

    const url =
      `http://${getHost()}/call` + `/topic/${selectedSchedule.topicKey}`;
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
          value: data.triggerAddress || "localhost:4000",
        },
      ]),
    };

    const startTime = window.performance.now();
    const res = await fetch(url, requestOptions);

    const callResponse = await generateResponse(res, startTime);
    setResponse(callResponse);

    setTimeout(() => setCallLoading(false), 300);
  };

  return (
    <div className="flex max-w-7xl flex-col md:flex-row gap-8 md:pr-8">
      <div className="w-full md:w-7/12 flex flex-col gap-8">
        <h2 className="text-2xl font-medium text-blue-900">
          Schedule - {selectedSchedule?.topicKey}
        </h2>
        <div>
          <nav className="flex items-end gap-4" aria-label="Breadcrumb">
            <ol role="list" className="flex w-11/12 items-center gap-4">
              <li className="w-full">
                <Select<Schedule>
                  id="topic-select"
                  items={data.schedules}
                  label="Topic"
                  selected={selectedSchedule}
                  setSelected={setSelectedSchedule}
                  display={(v) => (
                    <div className="flex items-center p-0.5 text-lg gap-4">
                      {v?.workerKey}
                    </div>
                  )}
                />
              </li>
            </ol>
            <div className="ml-auto">
              <button
                type="button"
                data-testid="trigger-topic-btn"
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
                      href={`http://${data.triggerAddress}/topic/${selectedSchedule?.topicKey}`}
                      target="_blank"
                    >
                      http://{data.triggerAddress}/topic/
                      {selectedSchedule?.topicKey}
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
                    <Badge status={response.status >= 400 ? "red" : "green"}>
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
                            count: Object.keys(response.headers || {}).length,
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
    </div>
  );
};

export default ScheduleExplorer;
