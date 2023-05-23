import { Fragment, PropsWithChildren, ReactNode, useState } from "react";
import { Dialog, Transition } from "@headlessui/react";
import {
  ArrowTopRightOnSquareIcon,
  Bars3Icon,
  GlobeAltIcon,
  XMarkIcon,
  ClockIcon,
  ChatBubbleLeftIcon,
  CircleStackIcon,
  MegaphoneIcon,
} from "@heroicons/react/24/outline";
import classNames from "classnames";
import { useWebSocket } from "../../lib/hooks/use-web-socket";
import { Toaster } from "react-hot-toast";

const resourceLinks = [
  {
    name: "Nitric Docs",
    href: "https://nitric.io/docs",
    icon: ArrowTopRightOnSquareIcon,
  },
  {
    name: "GitHub",
    href: "https://github.com/nitrictech/nitric",
    icon: ArrowTopRightOnSquareIcon,
  },
  {
    name: "Join us on Discord",
    href: "https://discord.gg/Webemece5C",
    icon: ArrowTopRightOnSquareIcon,
  },
  {
    name: "Send Feedback",
    href: "https://github.com/nitrictech/nitric/discussions/new?category=general&title=Local%20Dashboard%20Feedback",
    icon: ChatBubbleLeftIcon,
  },
];

interface Props extends PropsWithChildren {
  title: string;
  routePath: string;
  secondLevelNav: ReactNode;
}

const AppLayout: React.FC<Props> = ({
  title = "Dev Dashboard",
  children,
  secondLevelNav,
  routePath = "/",
}) => {
  const { data } = useWebSocket();
  const [sidebarOpen, setSidebarOpen] = useState(false);

  // remove trailing slash
  routePath = routePath !== "/" ? routePath.replace(/\/$/, "") : routePath;

  const navigation = [
    {
      name: "API Explorer",
      href: "/",
      icon: GlobeAltIcon,
      count: data?.apis.length,
    },
    {
      name: "Schedules",
      href: "/schedules",
      icon: ClockIcon,
      count: data?.schedules?.length || 0,
    },
    {
      name: "Storage",
      href: "/storage",
      icon: CircleStackIcon,
      count: data?.buckets?.length || 0,
    },
    {
      name: "Topics",
      href: "/topics",
      icon: MegaphoneIcon,
      count: data?.topics?.length,
    },
    // { name: "Storage", href: "#", icon: CircleStackIcon, current: false },
    // { name: "Collections", href: "#", icon: FolderIcon, current: false },
    // { name: "Secrets", href: "#", icon: LockClosedIcon, current: false },
  ];

  return (
    <>
      <Toaster position="top-right" />
      <Transition.Root show={sidebarOpen} as={Fragment}>
        <Dialog
          as="div"
          className="relative z-50 lg:hidden"
          onClose={setSidebarOpen}
        >
          <Transition.Child
            as={Fragment}
            enter="transition-opacity ease-linear duration-300"
            enterFrom="opacity-0"
            enterTo="opacity-100"
            leave="transition-opacity ease-linear duration-300"
            leaveFrom="opacity-100"
            leaveTo="opacity-0"
          >
            <div className="fixed inset-0 bg-white/80" />
          </Transition.Child>

          <div className="fixed inset-0 flex">
            <Transition.Child
              as={Fragment}
              enter="transition ease-in-out duration-300 transform"
              enterFrom="-translate-x-full"
              enterTo="translate-x-0"
              leave="transition ease-in-out duration-300 transform"
              leaveFrom="translate-x-0"
              leaveTo="-translate-x-full"
            >
              <Dialog.Panel className="relative mr-16 flex w-full max-w-xs flex-1">
                <Transition.Child
                  as={Fragment}
                  enter="ease-in-out duration-300"
                  enterFrom="opacity-0"
                  enterTo="opacity-100"
                  leave="ease-in-out duration-300"
                  leaveFrom="opacity-100"
                  leaveTo="opacity-0"
                >
                  <div className="absolute top-0 left-full flex w-16 justify-center pt-5">
                    <button
                      type="button"
                      className="-m-2.5 p-2.5"
                      onClick={() => setSidebarOpen(false)}
                    >
                      <span className="sr-only">Close sidebar</span>
                      <XMarkIcon
                        className="h-6 w-6 text-white"
                        aria-hidden="true"
                      />
                    </button>
                  </div>
                </Transition.Child>
                {/* Sidebar component, swap this element with another sidebar if you like */}
                <div className="flex grow flex-col gap-y-5 overflow-y-auto bg-white px-6 pb-2">
                  <div className="flex h-16 shrink-0 items-center">
                    <img
                      className="h-8 w-auto"
                      src="/nitric-no-text.svg"
                      alt="Nitric Logo"
                    />
                  </div>
                  <nav className="flex flex-1 flex-col">
                    <ul className="flex flex-1 flex-col gap-y-7">
                      <li>
                        <ul className="-mx-2 space-y-1">
                          {navigation.map((item) => (
                            <li key={item.name}>
                              <a
                                href={item.href}
                                className={classNames(
                                  item.href === routePath
                                    ? "bg-gray-50 text-blue-600"
                                    : "text-gray-700 hover:text-blue-600 hover:bg-gray-50",
                                  "group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold"
                                )}
                              >
                                <item.icon
                                  className={classNames(
                                    item.href === routePath
                                      ? "text-blue-600"
                                      : "text-gray-400 group-hover:text-blue-600",
                                    "h-6 w-6 shrink-0"
                                  )}
                                  aria-hidden="true"
                                />
                                {item.name}
                              </a>
                            </li>
                          ))}
                        </ul>
                      </li>
                      <li>
                        <div className="text-xs font-semibold leading-6 text-gray-400">
                          Resources & Feedback
                        </div>
                        <ul className="-mx-2 mt-2 space-y-1">
                          {resourceLinks.map((link) => (
                            <li key={link.name}>
                              <a
                                href={link.href}
                                className={classNames(
                                  "text-gray-700 hover:text-blue-600 items-center hover:bg-gray-50",
                                  "group flex gap-x-3 rounded-md p-2 text-sm leading-6 font-semibold"
                                )}
                              >
                                <span className="truncate">{link.name}</span>
                                <link.icon className="w-4 h-4" />
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

      <div className="hidden lg:fixed border-r lg:inset-y-0 lg:left-0 lg:z-50 lg:block lg:w-20 lg:overflow-y-auto lg:bg-white lg:pb-4">
        <div className="flex h-16 shrink-0 items-center justify-center">
          <img
            className="h-8 w-auto"
            src="/nitric-no-text.svg"
            alt="Nitric Logo"
          />
        </div>
        <nav className="mt-8">
          <ul className="flex flex-col items-center space-y-1">
            {navigation.map((item) => (
              <li key={item.name}>
                <a
                  href={item.href}
                  className={classNames(
                    item.href === routePath
                      ? "bg-gray-100 text-blue-600"
                      : "text-gray-400 hover:text-blue-600 hover:bg-gray-100",
                    "group relative flex gap-x-3 rounded-md p-3 text-sm leading-6 font-semibold"
                  )}
                >
                  <item.icon className="h-6 w-6 shrink-0" aria-hidden="true" />
                  <span className="sr-only">{item.name}</span>
                  {item.count ? (
                    <span
                      data-testid={`${item.name}-count`}
                      className="absolute right-0 bottom-0 flex items-center justify-center h-4 w-4 text-xs -translate-y-1/2 translate-x-1/2 transform rounded-full bg-white ring-2 ring-gray-100"
                    >
                      {item.count}
                    </span>
                  ) : null}
                </a>
              </li>
            ))}
          </ul>
        </nav>
      </div>

      <div className="sticky top-0 z-40 flex items-center gap-x-6 bg-white px-4 py-4 shadow-sm sm:px-6 lg:hidden">
        <button
          type="button"
          className="-m-2.5 p-2.5 text-gray-400 lg:hidden"
          onClick={() => setSidebarOpen(true)}
        >
          <span className="sr-only">Open sidebar</span>
          <Bars3Icon className="h-6 w-6" aria-hidden="true" />
        </button>
        <div className="flex-1 text-sm font-semibold leading-6 text-blue-600">
          Nitric Dashboard / {title}
        </div>
      </div>

      <aside className="fixed inset-y-0 left-20 pt-20 hidden w-72 overflow-y-auto border-r border-gray-200 py-6 xl:block">
        {secondLevelNav}
      </aside>

      <main className="lg:pl-20">
        <div className="xl:pl-80">
          <div className="px-4 py-10 sm:px-6 lg:px-8 lg:py-12">
            <h1 className="text-4xl text-blue-800 font-bold mb-12">{title}</h1>
            {children}
          </div>
        </div>
      </main>
    </>
  );
};

export default AppLayout;
