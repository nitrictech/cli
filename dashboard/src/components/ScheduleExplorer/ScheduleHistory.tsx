import type { Schedule, ScheduleHistoryItem } from "../../types";
import Badge from "../shared/Badge";
import { getDateString } from "../../lib/utils";

interface Props {
  history: ScheduleHistoryItem[];
  setSelectedSchedule: (schedule: Schedule) => void;
}

const ScheduleHistory: React.FC<Props> = ({ history, setSelectedSchedule }) => {
  if (!history.length) {
    return <p>There is no history.</p>;
  }

  return (
    <div className="pb-10">
      <div className="flex flex-col gap-2 overflow-y-scroll max-h-[40rem]">
        {history
          .sort((a, b) => b.time - a.time)
          .map((h) => (
            <div
              onClick={() => setSelectedSchedule(h.schedule)}
              className="flex flex-col gap-2 p-4 border border-slate-200 hover:bg-slate-100 rounded-lg hover:cursor-pointer"
            >
              <div className="flex flex-row justify-between">
                <div className="flex flex-row gap-4">
                  <Badge status={h.success ? "green" : "red"}>
                    {h.success ? "success" : "failure"}
                  </Badge>
                </div>
                <p>{getDateString(h.time)}</p>
              </div>
              <p>{h.schedule.topicKey}</p>
            </div>
          ))}
      </div>
    </div>
  );
};

export default ScheduleHistory;
