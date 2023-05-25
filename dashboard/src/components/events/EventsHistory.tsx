import type { EventHistoryItem, WorkerResource } from "../../types";
import Badge from "../shared/Badge";
import { formatJSON, getDateString } from "../../lib/utils";
import { Disclosure } from "@headlessui/react";
import { ChevronUpIcon } from "@heroicons/react/20/solid";
import CodeEditor from "../apis/CodeEditor";

interface Props {
  history: EventHistoryItem[];
  selectedWorker: WorkerResource;
}

const EventsHistory: React.FC<Props> = ({ selectedWorker, history }) => {
  const requestHistory = history
    .sort((a, b) => b.time - a.time)
    .filter((h) => h.event)
    .filter((h) => h.event.topicKey && selectedWorker.topicKey);

  if (!requestHistory.length) {
    return <p>There is no history.</p>;
  }

  return (
    <div className="pb-10">
      <div className="flex flex-col gap-2 overflow-y-scroll max-h-[40rem]">
        {requestHistory.map((h, idx) => (
          <EventHistoryAccordion key={idx} {...h} />
        ))}
      </div>
    </div>
  );
};

const EventHistoryAccordion: React.FC<EventHistoryItem> = ({
  event,
  payload,
  success,
  time,
}) => {
  const formattedPayload = payload ? formatJSON(payload) : "";

  return (
    <Disclosure>
      {({ open }) => (
        <>
          <Disclosure.Button className="flex w-full justify-between rounded-lg bg-white border border-slate-100 px-4 py-2 text-left text-sm font-medium text-black hover:bg-blue-100 focus:outline-none focus-visible:ring focus-visible:ring-blue-500 focus-visible:ring-opacity-75">
            <div className="flex flex-row justify-between w-full">
              <div className="flex flex-row gap-4 w-2/3">
                <Badge
                  status={success ? "green" : "red"}
                  className="!text-md w-16 h-6"
                >
                  {success ? "success" : "failure"}
                </Badge>
                <p className="text-ellipsis">{event.topicKey}</p>
              </div>
              <div className="flex flex-row gap-4">
                <p>{getDateString(time)}</p>
                {payload && (
                  <ChevronUpIcon
                    className={`${
                      open ? "rotate-180 transform" : ""
                    } h-5 w-5 text-blue-500`}
                  />
                )}
              </div>
            </div>
          </Disclosure.Button>

          {payload && (
            <Disclosure.Panel className="px-4 pt-4 pb-2 text-sm text-gray-500">
              <div className="flex flex-col gap-8">
                <div className="flex flex-col gap-2">
                  <p className="text-md font-semibold">Payload</p>
                  <CodeEditor
                    contentType="application/json"
                    readOnly={true}
                    value={formattedPayload}
                    title="Payload"
                  />
                </div>
              </div>
            </Disclosure.Panel>
          )}
        </>
      )}
    </Disclosure>
  );
};

export default EventsHistory;
