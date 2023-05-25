import type { FC } from "react";
import { Badge } from "../shared";
import type { Method } from "../../types";

interface Props {
  method: Method;
  className?: string;
}

export const APIMethodBadge: FC<Props> = ({ method, className }) => {
  return (
    <Badge
      className={className}
      status={
        (
          {
            DELETE: "red",
            POST: "green",
            PUT: "yellow",
            PATCH: "orange",
            GET: "blue",
          } as any
        )[method]
      }
    >
      {method}
    </Badge>
  );
};
