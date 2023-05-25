import classNames from "classnames";
import type {
  ForwardRefExoticComponent,
  InputHTMLAttributes,
  SVGProps,
} from "react";

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  hideLabel?: boolean;
  id: string;
  icon?: ForwardRefExoticComponent<SVGProps<SVGSVGElement>>;
}

const TextField = ({
  label,
  hideLabel,
  id,
  icon: Icon,
  ...inputProps
}: Props) => {
  return (
    <div>
      <label
        htmlFor={id}
        className={
          hideLabel
            ? "sr-only"
            : "block text-sm font-medium leading-6 text-gray-900"
        }
      >
        {label}
      </label>
      <div className="relative mt-2 rounded-md shadow-sm">
        {Icon && (
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
            <Icon className="h-5 w-5 text-gray-400" aria-hidden="true" />
          </div>
        )}
        <input
          {...inputProps}
          id={id}
          className={classNames(
            "block w-full px-2 rounded-md border-0 py-1.5 text-gray-900 ring-1 ring-inset ring-gray-300 placeholder:text-gray-400 focus:ring-2 focus:ring-inset focus:ring-blue-600 sm:text-sm sm:leading-6",
            Icon && "pl-10"
          )}
        />
      </div>
    </div>
  );
};

export default TextField;
