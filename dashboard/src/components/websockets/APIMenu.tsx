import { Menu, Transition } from "@headlessui/react";
import { EllipsisHorizontalIcon } from "@heroicons/react/20/solid";
import classNames from "classnames";
import { Fragment } from "react";
import type { Endpoint } from "../../types";
import { LOCAL_STORAGE_KEY } from "./WSExplorer";
import { useHistory } from "../../lib/hooks/use-history";
import { formatJSON } from "../../lib/utils";

interface Props {
  selected: Endpoint;
  onAfterClear: () => void;
}

const APIMenu: React.FC<Props> = ({ selected, onAfterClear }) => {
  const { deleteHistory } = useHistory("apis");
  const clearHistory = async () => {
    const prefix = `${LOCAL_STORAGE_KEY}-${selected.api}-`;

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);
      if (key?.startsWith(prefix)) {
        localStorage.removeItem(key);
      }
    }

    localStorage.removeItem(`${LOCAL_STORAGE_KEY}-requests`);

    await deleteHistory();

    onAfterClear();
  };

  const downloadSpec = () => {
    const json = formatJSON(selected.doc);
    const blob = new Blob([json], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${selected.api}-spec.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
  };

  return (
    <Menu as="div" className="relative ml-auto">
      <Menu.Button className="-m-2.5 block p-2.5 text-gray-400 hover:text-gray-500">
        <span className="sr-only">Open options</span>
        <EllipsisHorizontalIcon
          className="h-10 w-10 md:h-6 md:w-6"
          aria-hidden="true"
        />
      </Menu.Button>
      <Transition
        as={Fragment}
        enter="transition ease-out duration-100"
        enterFrom="transform opacity-0 scale-95"
        enterTo="transform opacity-100 scale-100"
        leave="transition ease-in duration-75"
        leaveFrom="transform opacity-100 scale-100"
        leaveTo="transform opacity-0 scale-95"
      >
        <Menu.Items className="absolute right-0 z-10 mt-0.5 w-40 origin-top-right rounded-md bg-white py-2 shadow-lg ring-1 ring-gray-900/5 focus:outline-none">
          <Menu.Item>
            {({ active }) => (
              <button
                className={classNames(
                  active ? "bg-gray-50" : "",
                  "flex px-3 py-1 w-full text-sm leading-6 text-gray-900"
                )}
                onClick={downloadSpec}
              >
                Export Spec
              </button>
            )}
          </Menu.Item>
          <Menu.Item>
            {({ active }) => (
              <button
                onClick={clearHistory}
                className={classNames(
                  active ? "bg-gray-50" : "",
                  "flex px-3 py-1 w-full text-sm leading-6 text-gray-900"
                )}
              >
                Clear History
              </button>
            )}
          </Menu.Item>
        </Menu.Items>
      </Transition>
    </Menu>
  );
};

export default APIMenu;
