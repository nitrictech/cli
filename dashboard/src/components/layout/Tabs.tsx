import classNames from "classnames";

interface Tab {
  name: string;
  count?: number;
}

interface Props {
  tabs: Tab[];
  index: number;
  setIndex: React.Dispatch<React.SetStateAction<number>>;
  round?: boolean;
}

const Tabs: React.FC<Props> = ({ tabs, index, setIndex, round }) => {
  return (
    <div>
      <div className="sm:hidden">
        <label htmlFor="tabs" className="sr-only">
          Select a tab
        </label>
        {/* Use an "onChange" listener to redirect the user to the selected tab URL. */}
        <select
          id="tabs"
          name="tabs"
          className="block w-full rounded-md border-gray-300 focus:border-blue-500 focus:ring-blue-500"
          defaultValue={tabs[index].name}
          onChange={(e) => setIndex(parseInt(e.target.value))}
        >
          {tabs.map((tab, idx) => (
            <option key={tab.name} value={idx}>
              {tab.name}
            </option>
          ))}
        </select>
      </div>
      <div className="hidden sm:block">
        <nav
          className={classNames(
            "isolate flex divide-x divide-gray-200 shadow",
            round && "rounded-lg"
          )}
          aria-label="Tabs"
        >
          {tabs.map((tab, tabIdx) => (
            <button
              key={tab.name}
              onClick={() => setIndex(tabIdx)}
              data-testid={`${tab.name}-tab-btn`}
              className={classNames(
                tabIdx === index
                  ? "text-gray-900"
                  : "text-gray-500 hover:text-gray-700",
                tabIdx === 0 ? (round ? "rounded-l-lg" : "rounded-tl-lg") : "",
                tabIdx === tabs.length - 1
                  ? round
                    ? "rounded-r-lg"
                    : "rounded-tr-lg"
                  : "",
                "group relative min-w-0 flex-1 overflow-hidden bg-white py-4 px-4 text-center text-sm font-medium hover:bg-gray-50 focus:z-10"
              )}
              aria-current={tabIdx === index ? "page" : undefined}
            >
              <span>{tab.name}</span>
              {tab.count ? (
                <span
                  className={classNames(
                    tabIdx === index
                      ? "bg-indigo-100 text-blue-600"
                      : "bg-gray-100 text-gray-900",
                    "ml-3 hidden rounded-full py-0.5 px-2.5 text-xs font-medium md:inline-block"
                  )}
                >
                  {tab.count}
                </span>
              ) : null}
              <span
                aria-hidden="true"
                className={classNames(
                  tabIdx === index ? "bg-blue-500" : "bg-transparent",
                  "absolute inset-x-0 bottom-0 h-0.5"
                )}
              />
            </button>
          ))}
        </nav>
      </div>
    </div>
  );
};

export default Tabs;
