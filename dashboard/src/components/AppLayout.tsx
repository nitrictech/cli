import { Fragment, PropsWithChildren, useState } from "react";
import { Dialog, Transition } from "@headlessui/react";
import {
  ArrowTopRightOnSquareIcon,
  Bars3Icon,
  CircleStackIcon,
  ClockIcon,
  FolderIcon,
  HomeIcon,
  LockClosedIcon,
  MagnifyingGlassIcon,
  MegaphoneIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import classNames from "classnames";

const navigation = [
  { name: "Dashboard", href: "#", icon: HomeIcon, current: true },
  {
    name: "API Explorer",
    href: "#",
    icon: MagnifyingGlassIcon,
    current: false,
  },
  // { name: "Schedules", href: "#", icon: ClockIcon, current: false },
  // { name: "Storage", href: "#", icon: CircleStackIcon, current: false },
  // { name: "Collections", href: "#", icon: FolderIcon, current: false },
  // { name: "Messages", href: "#", icon: MegaphoneIcon, current: false },
  // { name: "Secrets", href: "#", icon: LockClosedIcon, current: false },
];
const helpLinks = [
  {
    name: "Nitric Docs",
    href: "https://nitric.io/docs",
  },
  {
    name: "GitHub",
    href: "https://github.com/nitrictech/nitric",
  },
  {
    name: "Join us on Discord",
    href: "https://discord.gg/Webemece5C",
  },
];

const AppLayout: React.FC<PropsWithChildren> = ({ children }) => {
  const [sidebarOpen, setSidebarOpen] = useState(false);

  return (
    <>
      <Transition.Root show={sidebarOpen} as={Fragment}>
        <Dialog
          as='div'
          className='relative z-50 lg:hidden'
          onClose={setSidebarOpen}
        >
          <Transition.Child
            as={Fragment}
            enter='transition-opacity ease-linear duration-300'
            enterFrom='opacity-0'
            enterTo='opacity-100'
            leave='transition-opacity ease-linear duration-300'
            leaveFrom='opacity-100'
            leaveTo='opacity-0'
          >
            <div className='fixed inset-0 bg-gray-900/80' />
          </Transition.Child>

          <div className='fixed inset-0 flex'>
            <Transition.Child
              as={Fragment}
              enter='transition ease-in-out duration-300 transform'
              enterFrom='-translate-x-full'
              enterTo='translate-x-0'
              leave='transition ease-in-out duration-300 transform'
              leaveFrom='translate-x-0'
              leaveTo='-translate-x-full'
            >
              <Dialog.Panel className='relative mr-16 flex w-full max-w-xs flex-1'>
                <Transition.Child
                  as={Fragment}
                  enter='ease-in-out duration-300'
                  enterFrom='opacity-0'
                  enterTo='opacity-100'
                  leave='ease-in-out duration-300'
                  leaveFrom='opacity-100'
                  leaveTo='opacity-0'
                >
                  <div className='absolute top-0 left-full flex w-16 justify-center pt-5'>
                    <button
                      type='button'
                      className='-m-2.5 p-2.5'
                      onClick={() => setSidebarOpen(false)}
                    >
                      <span className='sr-only'>Close sidebar</span>
                      <XMarkIcon
                        className='h-6 w-6 text-white'
                        aria-hidden='true'
                      />
                    </button>
                  </div>
                </Transition.Child>
                {/* Sidebar component, swap this element with another sidebar if you like */}
                <div className='flex grow flex-col gap-y-5 overflow-y-auto bg-white px-6 pb-2'>
                  <div className='flex h-16 shrink-0 items-center'>
                    <img
                      className='h-8 w-auto'
                      src='./nitric-no-text.svg'
                      alt='Nitric Logo'
                    />
                  </div>
                  <nav className='flex flex-1 flex-col'>
                    <ul role='list' className='flex flex-1 flex-col gap-y-7'>
                      <li>
                        <ul role='list' className='-mx-2 space-y-1'>
                          {navigation.map((item) => (
                            <li key={item.name}>
                              <a
                                href={item.href}
                                className={classNames(
                                  item.current
                                    ? "bg-gray-50 text-blue-600"
                                    : "text-gray-700 hover:text-blue-600 hover:bg-gray-50",
                                  "group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold"
                                )}
                              >
                                <item.icon
                                  className={classNames(
                                    item.current
                                      ? "text-blue-600"
                                      : "text-gray-400 group-hover:text-blue-600",
                                    "h-6 w-6 shrink-0"
                                  )}
                                  aria-hidden='true'
                                />
                                {item.name}
                              </a>
                            </li>
                          ))}
                        </ul>
                      </li>
                      <li>
                        <div className='text-xs font-semibold leading-6 text-gray-400'>
                          Resources
                        </div>
                        <ul role='list' className='-mx-2 mt-2 space-y-1'>
                          {helpLinks.map((link) => (
                            <li key={link.name}>
                              <a
                                href={link.href}
                                className={classNames(
                                  "text-gray-700 hover:text-blue-600 hover:bg-gray-50",
                                  "group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold"
                                )}
                              >
                                <span className='truncate'>{link.name}</span>
                              </a>
                            </li>
                          ))}
                        </ul>
                      </li>
                    </ul>
                  </nav>
                </div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </Dialog>
      </Transition.Root>

      {/* Static sidebar for desktop */}
      <div className='hidden lg:fixed lg:inset-y-0 lg:z-50 lg:flex lg:w-72 lg:flex-col'>
        {/* Sidebar component, swap this element with another sidebar if you like */}
        <div className='flex grow flex-col gap-y-5 overflow-y-auto border-r border-gray-200 bg-white px-6'>
          <div className='flex h-16 shrink-0 items-center'>
            <img
              className='h-8 w-auto'
              src='./nitric-no-text.svg'
              alt='Nitric Logo'
            />
          </div>
          <nav className='flex flex-1 flex-col'>
            <ul role='list' className='flex flex-1 flex-col gap-y-7'>
              <li>
                <ul role='list' className='-mx-2 space-y-1'>
                  {navigation.map((item) => (
                    <li key={item.name}>
                      <a
                        href={item.href}
                        className={classNames(
                          item.current
                            ? "bg-gray-50 text-blue-600"
                            : "text-gray-700 hover:text-blue-600 hover:bg-gray-50",
                          "group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold"
                        )}
                      >
                        <item.icon
                          className={classNames(
                            item.current
                              ? "text-blue-600"
                              : "text-gray-400 group-hover:text-blue-600",
                            "h-6 w-6 shrink-0"
                          )}
                          aria-hidden='true'
                        />
                        {item.name}
                      </a>
                    </li>
                  ))}
                </ul>
              </li>
              <li>
                <div className='text-xs font-semibold leading-6 text-gray-400'>
                  Resources
                </div>
                <ul role='list' className='-mx-2 mt-2 space-y-1'>
                  {helpLinks.map((link) => (
                    <li key={link.name}>
                      <a
                        href={link.href}
                        target='_blank'
                        rel='noreferrer'
                        className={classNames(
                          "text-gray-700 hover:text-blue-600 hover:bg-gray-50",
                          "group flex gap-x-2 rounded-md p-2 items-center text-sm leading-6 font-semibold"
                        )}
                      >
                        <span className='truncate'>{link.name}</span>
                        <ArrowTopRightOnSquareIcon className='w-4 h-4' />
                      </a>
                    </li>
                  ))}
                </ul>
              </li>
            </ul>
          </nav>
        </div>
      </div>

      <div className='sticky top-0 z-40 flex items-center gap-x-6 bg-white py-4 px-4 shadow-sm sm:px-6 lg:hidden'>
        <button
          type='button'
          className='-m-2.5 p-2.5 text-gray-700 lg:hidden'
          onClick={() => setSidebarOpen(true)}
        >
          <span className='sr-only'>Open sidebar</span>
          <Bars3Icon className='h-6 w-6' aria-hidden='true' />
        </button>
        <div className='flex-1 text-sm font-semibold leading-6 text-gray-900'>
          Dev Dashboard
        </div>
      </div>

      <main className='py-10 lg:pl-72'>
        <div className='px-4 sm:px-6 lg:px-8'>{children}</div>
      </main>
    </>
  );
};

export default AppLayout;
