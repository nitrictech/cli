import type { EventHistoryItem, WorkerResource } from "../../types";
import Badge from "../shared/Badge";
import { getDateString } from "../../lib/utils";
import { useHistory } from "../../lib/hooks/use-history";

interface Props {
  history: EventHistoryItem[];
  setSelectedWorker: (worker: WorkerResource) => void;
}

const EventsHistory: React.FC<Props> = ({ setSelectedWorker, history }) => {
  return (
    <div className="pb-10">
      <div className="flex flex-col gap-2 overflow-y-scroll max-h-[40rem]">
        {history
          .sort((a, b) => b.time - a.time)
          .map((h, idx) => (
            <button
              key={idx}
              aria-label={`selected-request-${idx}`}
              onClick={() => setSelectedWorker(h.event)}
              className="flex flex-col gap-2 p-4 border border-slate-200 hover:bg-slate-100 rounded-lg hover:cursor-pointer"
            >
              <div className="flex flex-row w-full justify-between items-center">
                <Badge status={h.success ? "green" : "red"}>
                  {h.success ? "success" : "failure"}
                </Badge>
                <p>{getDateString(h.time)}</p>
              </div>
              <p>{h.event.topicKey}</p>
            </button>
          ))}
      </div>
    </div>
  );
};

export default EventsHistory;
