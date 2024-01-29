import { cn } from "@/lib/utils";
import { FC, Fragment } from "react";

interface Group {
  name: string;
  rows: any[][];
}

interface Props {
  headers: any[];
  groups: Group[];
  rowDataClassName?: string;
}

const TableGroup: FC<Props> = ({ headers, groups, rowDataClassName }) => {
  return (
    <div className="-my-2 overflow-x-auto">
      <div className="inline-block min-w-full py-2 align-middle">
        <table className="min-w-full">
          <thead className="sr-only">
            <tr>
              {headers.map((header) => (
                <th key={header}>{header}</th>
              ))}
            </tr>
          </thead>
          <tbody className="bg-white">
            {groups.map((group) => (
              <Fragment key={group.name}>
                <tr className="border-t border-gray-200">
                  <th
                    colSpan={5}
                    scope="colgroup"
                    className="bg-gray-100 py-2 pl-4 pr-3 text-left text-sm font-semibold text-gray-900 sm:pl-3"
                  >
                    {group.name}
                  </th>
                </tr>
                {group.rows.map((row, rowIdx) => (
                  <tr
                    key={rowIdx}
                    className={cn(
                      rowIdx === 0 ? "border-gray-300" : "border-gray-200",
                      "border-t"
                    )}
                  >
                    {row.map((rowData, rowDataIdx) => (
                      <td
                        key={rowDataIdx}
                        title={rowData}
                        className={cn(
                          "truncate py-4 pl-4 pr-3 text-sm font-medium text-gray-500 sm:pl-3",
                          rowDataClassName
                        )}
                      >
                        {rowData}
                      </td>
                    ))}
                  </tr>
                ))}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};

export default TableGroup;
