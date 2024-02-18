import { cn } from '@/lib/utils'

interface Tab {
  name: string
  count?: number
}

interface Props {
  tabs: Tab[]
  index: number
  setIndex: React.Dispatch<React.SetStateAction<number>>
  round?: boolean
  pill?: boolean
}

const Tabs: React.FC<Props> = ({ tabs, index, setIndex, round, pill }) => {
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
          className={cn(
            pill
              ? 'flex space-x-4'
              : 'isolate flex divide-x divide-gray-200 shadow',
            round && 'rounded-lg',
          )}
          aria-label="Tabs"
        >
          {tabs.map((tab, tabIdx) => (
            <button
              key={tab.name}
              onClick={() => setIndex(tabIdx)}
              data-testid={`${tab.name}-tab-btn`}
              className={cn(
                tabIdx === index
                  ? pill
                    ? 'bg-gray-100 text-gray-700'
                    : 'text-gray-900'
                  : pill
                    ? 'text-gray-500 hover:text-gray-700'
                    : 'text-gray-500 hover:text-gray-700',
                tabIdx === 0 && !pill
                  ? round
                    ? 'rounded-l-lg'
                    : 'rounded-tl-lg'
                  : '',
                tabIdx === tabs.length - 1 && !pill
                  ? round
                    ? 'rounded-r-lg'
                    : 'rounded-tr-lg'
                  : '',
                pill
                  ? 'rounded-md px-3 py-2 text-sm font-medium'
                  : 'group relative min-w-0 flex-1 overflow-hidden bg-white px-4 py-4 text-center text-sm font-medium hover:bg-gray-50 focus:z-10',
              )}
              aria-current={tabIdx === index ? 'page' : undefined}
            >
              <span>{tab.name}</span>
              {tab.count ? (
                <span
                  className={cn(
                    tabIdx === index
                      ? 'bg-indigo-100 text-primary'
                      : 'bg-gray-100 text-gray-900',
                    'ml-3 hidden rounded-full px-2.5 py-0.5 text-xs font-medium md:inline-block',
                  )}
                >
                  {tab.count}
                </span>
              ) : null}
              {!pill && (
                <span
                  aria-hidden="true"
                  className={cn(
                    tabIdx === index ? 'bg-blue-500' : 'bg-transparent',
                    'absolute inset-x-0 bottom-0 h-0.5',
                  )}
                />
              )}
            </button>
          ))}
        </nav>
      </div>
    </div>
  )
}

export default Tabs
