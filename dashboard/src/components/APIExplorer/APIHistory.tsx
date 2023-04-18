import { CheckIcon, ExclamationTriangleIcon } from "@heroicons/react/20/solid";
import type { HistoryItem } from "../../types";
import Badge from "../shared/Badge";

interface Props {
  history: HistoryItem[];
}

const APIHistory: React.FC<Props> = ({ history }) => {
  return (
    <div className="flow-root">
      <div className="mb-8 justify-between flex">
        <span className="font-medium">Request</span>
        <span className="font-medium">Duration</span>
      </div>
      <ul role="list" className="-mb-8">
        {history.map(({ time, request, response }, idx) => (
          <li key={time}>
            <div className="relative pb-8">
              {idx !== history.length - 1 ? (
                <span
                  className="absolute left-4 top-4 -ml-px h-full w-0.5 bg-gray-200"
                  aria-hidden="true"
                />
              ) : null}
              <div className="relative flex space-x-3">
                <div>
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
                <div className="flex min-w-0 flex-1 justify-between space-x-4 pt-1.5">
                  <div>
                    <p className="text-sm text-gray-500">{request.path} </p>
                  </div>
                  <div className="whitespace-nowrap text-right text-sm text-gray-500">
                    {response.time}
                  </div>
                </div>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default APIHistory;
