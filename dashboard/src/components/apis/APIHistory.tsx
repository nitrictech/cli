import type { RequestHistory, ApiHistoryItem } from "../../types";
import Badge from "../shared/Badge";
import type { FieldRow } from "../shared/FieldRows";
import { getDateString } from "../../lib/utils";

interface Props {
  history: ApiHistoryItem[];
  setSelectedRequest: (api: string, request: RequestHistory) => void;
}

const stringifyQueryParams = (queryParams: FieldRow[]) => {
  if (!queryParams) {
    return "";
  }
  if (queryParams.filter((p) => p.key && p.value).length === 0) {
    return "";
  }

  return `?${queryParams.map(
    (p, idx) =>
      `${p.key}=${p.value}` + (idx !== queryParams.length - 1 ? "&" : "")
  )}`;
};

const APIHistory: React.FC<Props> = ({ history, setSelectedRequest }) => {
  if (!history.length) {
    return <p>There is no history.</p>;
  }

  return (
    <div className="pb-10">
      <div className="flex flex-col gap-2 overflow-y-scroll max-h-[40rem]">
        {history
          .sort((a, b) => b.time - a.time)
          .filter((h) => h.request && h.response)
          .map((h, idx) => (
            <button
              key={idx}
              aria-label={`selected-request-${idx}`}
              onClick={() => setSelectedRequest(h.api, h.request)}
              className="flex flex-col gap-2 p-4 border border-slate-200 hover:bg-slate-100 rounded-lg hover:cursor-pointer"
            >
              <div className="flex flex-row justify-between w-full items-center">
                <div className="flex flex-row gap-2">
                  {h.response.status && (
                    <Badge status={h.response.status >= 400 ? "red" : "green"}>
                      {h.response.status}
                    </Badge>
                  )}
                  {h.request.method && (
                    <Badge
                      status={
                        (
                          {
                            DELETE: "red",
                            POST: "green",
                            PUT: "yellow",
                            GET: "blue",
                          } as any
                        )[h.request.method]
                      }
                      className="!text-md"
                    >
                      {h.request.method}
                    </Badge>
                  )}
                </div>
                <p>{getDateString(h.time)}</p>
              </div>
              <p>
                {h.api.replace("https://", "").replace("http://", "")}
                {h.request.path}
                {stringifyQueryParams(h.request.queryParams)}
              </p>
            </button>
          ))}
      </div>
    </div>
  );
};

export default APIHistory;
