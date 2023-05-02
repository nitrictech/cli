import { useEffect, useState } from "react";
import { useWebSocket } from "../lib/use-web-socket";
import Select from "./shared/Select";
import type { APIResponse, BucketNotification, Schedule } from "../types";
import Badge from "./shared/Badge";
import Spinner from "./shared/Spinner";
import { formatFileSize } from "./APIExplorer/format-file-size";
import Tabs from "./layout/Tabs";
import APIResponseContent from "./APIExplorer/APIResponseContent";
import { fieldRowArrToHeaders, getHost } from "../lib/utils";
import { generateResponse } from "../lib/generate-response";
import { formatResponseTime } from "./APIExplorer/format-response-time";
import Loading from "./shared/Loading";

export const LOCAL_STORAGE_KEY =
  "nitric-local-dash-bucket-notification-history";

const getId = (b: BucketNotification) =>
  b.bucket + b.notificationType + b.notificationPrefixFilter.replace("/", "");

const BucketNotificationExplorer = () => {
  const { data, loading } = useWebSocket();
  const [callLoading, setCallLoading] = useState(false);

  const [response, setResponse] = useState<APIResponse>();

  const [selectedBucket, setSelectedBucket] = useState<string>();
  const [selectedBucketNotification, setSelectedBucketNotification] =
    useState<BucketNotification>();
  const [responseTabIndex, setResponseTabIndex] = useState(0);

  useEffect(() => {
    if (data?.bucketNotifications?.length) {
      // restore history or select first if not selected
      if (!selectedBucketNotification) {
        const previousId = localStorage.getItem(
          `${LOCAL_STORAGE_KEY}-last-bucket-notification`
        );

        const bucketNotification =
          (previousId &&
            data.bucketNotifications.find((b) => getId(b) === previousId)) ||
          data.bucketNotifications[0];

        setSelectedBucketNotification(bucketNotification);
        setSelectedBucket(bucketNotification.bucket);
      } else {
        // could be a refresh from ws, so update the selected endpoint
        const latest = data.bucketNotifications.find(
          (b) => getId(b) === getId(selectedBucketNotification)
        );

        if (latest) {
          setSelectedBucketNotification(latest);
          setSelectedBucket(latest.bucket);
        }
      }
    }
  }, [data]);

  const handleSend = async (
    e: React.MouseEvent<HTMLButtonElement, MouseEvent>
  ) => {
    if (!selectedBucketNotification) return;
    setCallLoading(true);
    e.preventDefault();

    const url =
      `http://${getHost()}/call` +
      `/notification/bucket/${selectedBucketNotification.bucket}`;
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
      body: JSON.stringify({
        key: selectedBucketNotification.notificationPrefixFilter,
        type: selectedBucketNotification.notificationType,
      }),
    };

    const startTime = window.performance.now();
    const res = await fetch(url, requestOptions);

    const callResponse = await generateResponse(res, startTime);
    setResponse(callResponse);

    setTimeout(() => setCallLoading(false), 300);
  };

  return (
    <Loading delay={400} conditionToShow={!loading}>
      {selectedBucketNotification && data ? (
        <div className="flex max-w-7xl flex-col md:flex-row gap-8 md:pr-8">
          <div className="w-full md:w-7/12 flex flex-col gap-8">
            <h2 className="text-2xl font-medium text-blue-900">
              Bucket Notification - {selectedBucket}
            </h2>
            <div>
              <nav className="flex items-end gap-4" aria-label="Breadcrumb">
                <ol className="flex w-11/12 items-center gap-4">
                  <li className="w-full">
                    {data.bucketNotifications && (
                      <Select<string>
                        id="bucket-select"
                        items={[
                          ...new Set(
                            data.bucketNotifications.map((b) => b.bucket)
                          ),
                        ]}
                        label="Bucket"
                        selected={selectedBucket}
                        setSelected={setSelectedBucket}
                        display={(v) => (
                          <div className="flex items-center p-0.5 text-lg gap-4">
                            {v}
                          </div>
                        )}
                      />
                    )}
                  </li>
                  <li className="w-full">
                    {data.bucketNotifications && (
                      <Select<BucketNotification>
                        id="notification-select"
                        items={data.bucketNotifications.filter(
                          (b) => selectedBucket === b.bucket
                        )}
                        label="Notifications"
                        selected={selectedBucketNotification}
                        setSelected={setSelectedBucketNotification}
                        display={(v) => (
                          <div className="flex items-center p-0.5 text-lg gap-4">
                            {v.notificationType} - {v.notificationPrefixFilter}
                          </div>
                        )}
                      />
                    )}
                  </li>
                </ol>

                <div className="ml-auto">
                  <button
                    type="button"
                    data-testid="trigger-notification-btn"
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
                          href={`http://${data.triggerAddress}/notification/bucket/${selectedBucketNotification?.bucket}`}
                          target="_blank"
                          rel="noreferrer"
                        >
                          http://{data.triggerAddress}/notification/bucket/
                          {selectedBucketNotification?.bucket}
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
        </div>
      ) : !data?.bucketNotifications?.length ? (
        <div>
          Please refer to our documentation on{" "}
          <a
            className="underline"
            target="_blank"
            href="https://nitric.io/docs/buckets#create-notifications"
            rel="noreferrer"
          >
            creating bucket notifications
          </a>{" "}
          as we are unable to find any existing bucket notifications.
        </div>
      ) : null}
    </Loading>
  );
};

export default BucketNotificationExplorer;
