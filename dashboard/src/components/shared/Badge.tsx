import classNames from "classnames";
import type { PropsWithChildren } from "react";

interface Props extends PropsWithChildren {
  status: "red" | "green" | "yellow" | "blue" | "default";
  className?: string;
}

const Badge: React.FC<Props> = ({
  status = "default",
  className,
  children,
}) => {
  return (
    <span
      className={classNames(
        "inline-flex justify-center items-center rounded-full px-2.5 py-0.5 text-xs font-medium",
        status === "red" && "text-red-800 bg-red-100",
        status === "green" && "text-green-800 bg-green-100",
        status === "yellow" && "text-yellow-800 bg-yellow-100",
        status === "blue" && "text-blue-800 bg-blue-100",
        status === "default" && "text-gray-800 bg-gray-100",
        className
      )}
    >
      {children}
    </span>
  );
};

export default Badge;
