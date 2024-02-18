import { Menu, Transition } from '@headlessui/react'
import { EllipsisHorizontalIcon } from '@heroicons/react/20/solid'
import { cn } from '@/lib/utils'
import { Fragment } from 'react'
import { useHistory } from '../../lib/hooks/use-history'

interface Props {
  storageKey: string
  workerType: string
  selected: string
  onAfterClear: () => void
}

const EventsMenu: React.FC<Props> = ({
  workerType,
  storageKey,
  selected,
  onAfterClear,
}) => {
  const { deleteHistory } = useHistory(workerType)

  const clearHistory = async () => {
    const prefix = `${storageKey}-${selected}-`

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (key?.startsWith(prefix)) {
        localStorage.removeItem(key)
      }
    }

    localStorage.removeItem(`${storageKey}-requests`)

    await deleteHistory()

    onAfterClear()
  }

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
                onClick={clearHistory}
                className={cn(
                  active ? 'bg-gray-50' : '',
                  'flex w-full px-3 py-1 text-sm leading-6 text-gray-900',
                )}
              >
                Clear History
              </button>
            )}
          </Menu.Item>
        </Menu.Items>
      </Transition>
    </Menu>
  )
}

export default EventsMenu
