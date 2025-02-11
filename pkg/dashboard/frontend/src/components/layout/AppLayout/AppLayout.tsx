import { type PropsWithChildren, type ReactNode } from 'react'
import {
  DocumentDuplicateIcon,
  Bars3Icon,
  GlobeAltIcon,
  ClockIcon,
  ArchiveBoxIcon,
  MegaphoneIcon,
  QuestionMarkCircleIcon,
  PaperAirplaneIcon,
  ChatBubbleBottomCenterIcon,
  ChatBubbleLeftRightIcon,
  MapIcon,
  HeartIcon,
  CircleStackIcon,
  LockClosedIcon,
  CpuChipIcon,
  WindowIcon,
} from '@heroicons/react/24/outline'
import { cn } from '@/lib/utils'
import { useWebSocket } from '../../../lib/hooks/use-web-socket'
import { Toaster } from 'react-hot-toast'
import { TooltipProvider } from '../../ui/tooltip'
import { Button } from '../../ui/button'
import { ExclamationCircleIcon } from '@heroicons/react/20/solid'
import { Alert, AlertDescription, AlertTitle } from '../../ui/alert'
import { Spinner } from '../../shared'
import { Popover, PopoverContent, PopoverTrigger } from '../../ui/popover'
import { Sheet, SheetContent, SheetTrigger } from '../../ui/sheet'
import NavigationBar from './NavigationBar'

const DiscordLogo: React.FC<React.SVGProps<SVGSVGElement>> = ({
  className,
}) => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    className={className}
    viewBox="0 0 127.14 96.36"
  >
    <g id="Discord_Logos" data-name="Discord Logos">
      <g
        id="Discord_Logo_-_Large_-_White"
        data-name="Discord Logo - Large - White"
      >
        <path
          fill="#5865f2"
          d="M107.7,8.07A105.15,105.15,0,0,0,81.47,0a72.06,72.06,0,0,0-3.36,6.83A97.68,97.68,0,0,0,49,6.83,72.37,72.37,0,0,0,45.64,0,105.89,105.89,0,0,0,19.39,8.09C2.79,32.65-1.71,56.6.54,80.21h0A105.73,105.73,0,0,0,32.71,96.36,77.7,77.7,0,0,0,39.6,85.25a68.42,68.42,0,0,1-10.85-5.18c.91-.66,1.8-1.34,2.66-2a75.57,75.57,0,0,0,64.32,0c.87.71,1.76,1.39,2.66,2a68.68,68.68,0,0,1-10.87,5.19,77,77,0,0,0,6.89,11.1A105.25,105.25,0,0,0,126.6,80.22h0C129.24,52.84,122.09,29.11,107.7,8.07ZM42.45,65.69C36.18,65.69,31,60,31,53s5-12.74,11.43-12.74S54,46,53.89,53,48.84,65.69,42.45,65.69Zm42.24,0C78.41,65.69,73.25,60,73.25,53s5-12.74,11.44-12.74S96.23,46,96.12,53,91.08,65.69,84.69,65.69Z"
        />
      </g>
    </g>
  </svg>
)

const resourceLinks = [
  {
    name: 'Nitric Docs',
    href: 'https://nitric.io/docs',
    icon: DocumentDuplicateIcon,
    description:
      'Unlock the power of knowledge! Dive into our docs for helpful tips, tricks, and all the information you need to make the most out of Nitric',
  },
  {
    name: 'Send Feedback',
    href: 'https://github.com/nitrictech/nitric/discussions/new?category=general&title=Local%20Dashboard%20Feedback',
    icon: PaperAirplaneIcon,
    description:
      'Help us improve! Your feedback is valuable in shaping our roadmap',
  },
]

const communityLinks = [
  {
    name: 'Join us on Discord',
    href: 'https://nitric.io/chat',
    icon: DiscordLogo,
  },
  {
    name: 'GitHub Discussions',
    href: 'https://github.com/nitrictech/nitric/discussions',
    icon: ChatBubbleBottomCenterIcon,
  },
  {
    name: 'Sponsor',
    href: 'https://github.com/sponsors/nitrictech',
    icon: HeartIcon,
  },
]

interface Props extends PropsWithChildren {
  title: string
  routePath: string
  secondLevelNav?: ReactNode
  mainClassName?: string
  hideTitle?: boolean
}

const AppLayout: React.FC<Props> = ({
  title = 'Local Dashboard',
  children,
  secondLevelNav,
  mainClassName,
  hideTitle,
  routePath = '/',
}) => {
  const { data, state } = useWebSocket()

  // remove trailing slash
  routePath = routePath !== '/' ? routePath.replace(/\/$/, '') : routePath

  const navigation = [
    {
      name: 'APIs',
      href: '/',
      icon: GlobeAltIcon,
    },
    {
      name: 'Batch Jobs',
      href: '/jobs',
      icon: CpuChipIcon,
    },
    {
      name: 'Databases',
      href: '/databases',
      icon: CircleStackIcon,
    },
    {
      name: 'Schedules',
      href: '/schedules',
      icon: ClockIcon,
    },
    {
      name: 'Storage',
      href: '/storage',
      icon: ArchiveBoxIcon,
    },
    {
      name: 'Topics',
      href: '/topics',
      icon: MegaphoneIcon,
    },
    {
      name: 'Secrets',
      href: '/secrets',
      icon: LockClosedIcon,
    },
    {
      name: 'Websockets',
      href: '/websockets',
      icon: ChatBubbleLeftRightIcon,
    },
    {
      name: 'Websites',
      href: '/websites',
      icon: WindowIcon,
    },
    // { name: "Key Value Stores", href: "#", icon: FolderIcon, current: false },
  ]

  const showAlert = data?.connected === false || state === 'error'

  return (
    <TooltipProvider>
      <Toaster position="top-right" />

      <NavigationBar navigation={navigation} routePath={routePath} />

      {secondLevelNav && (
        <aside
          className={cn(
            'fixed inset-y-0 left-20 hidden w-80 overflow-y-auto overflow-x-hidden border-r border-gray-200 pb-6 pt-20 lg:block',
            showAlert && 'lg:mt-24',
          )}
        >
          {secondLevelNav}
        </aside>
      )}

      <main className="lg:pl-20">
        <div className="sticky top-0 z-40 flex h-16 shrink-0 items-center gap-x-4 border-b border-gray-200 bg-white px-4 sm:gap-x-6 sm:px-6 lg:px-8">
          <Sheet>
            <SheetTrigger asChild>
              <Button size="icon" variant="ghost" className="lg:hidden">
                <span className="sr-only">Open sidebar</span>
                <Bars3Icon className="h-6 w-6" aria-hidden="true" />
              </Button>
            </SheetTrigger>
            <SheetContent side="left" className="sm:max-w-xs">
              <div className="flex grow flex-col gap-y-5 bg-white">
                <div className="flex h-16 shrink-0 items-center gap-x-4">
                  <img
                    className="h-8 w-auto"
                    src="/nitric-no-text.svg"
                    alt="Nitric Logo"
                  />
                  <span className="max-w-96 truncate font-semibold">
                    {data?.projectName}
                  </span>
                </div>
                <nav className="flex flex-1 flex-col space-y-2">
                  <ul className="flex flex-1 flex-col gap-y-7">
                    <li>
                      <ul className="-mx-2 space-y-1">
                        {[
                          {
                            name: 'Architecture',
                            href: '/architecture',
                            icon: MapIcon,
                          },
                          ...navigation,
                        ].map((item) => (
                          <li key={item.name}>
                            <a
                              href={item.href}
                              className={cn(
                                item.href === routePath
                                  ? 'bg-gray-50 text-primary'
                                  : 'text-gray-700 hover:bg-gray-50 hover:text-primary',
                                'group flex items-center gap-x-3 rounded-md p-2 text-sm font-semibold leading-6',
                              )}
                            >
                              <item.icon
                                className={cn(
                                  item.href === routePath
                                    ? 'text-primary'
                                    : 'text-gray-400 group-hover:text-primary',
                                  'h-6 w-6 shrink-0',
                                )}
                                aria-hidden="true"
                              />
                              <span>{item.name}</span>
                            </a>
                          </li>
                        ))}
                      </ul>
                    </li>
                    <li>
                      <div className="text-sm font-semibold leading-6 text-gray-400">
                        Resources & Feedback
                      </div>
                      <ul className="-mx-2 mt-2 space-y-1">
                        {[...resourceLinks, ...communityLinks].map((link) => (
                          <li key={link.name}>
                            <a
                              href={link.href}
                              target="_blank"
                              rel="noreferrer"
                              className={cn(
                                'items-center text-gray-700 hover:bg-gray-50 hover:text-primary',
                                'group flex gap-x-3 rounded-md p-2 text-sm font-semibold leading-6',
                              )}
                            >
                              <span className="truncate">{link.name}</span>
                              <link.icon className="h-4 w-4" />
                            </a>
                          </li>
                        ))}
                      </ul>
                    </li>
                  </ul>
                </nav>
              </div>
            </SheetContent>
          </Sheet>
          {/* Separator */}
          <div
            className="h-6 w-px bg-gray-900/10 lg:hidden"
            aria-hidden="true"
          />
          {data?.projectName && (
            <div className="flex items-center gap-6 font-semibold leading-6 text-gray-900 md:text-lg">
              <span className="hidden max-w-[300px] truncate lg:block">
                {data.projectName}
              </span>{' '}
              <span className="hidden text-gray-300 lg:block">/</span>{' '}
              <Button
                className={cn(
                  '/architecture' === routePath && 'bg-accent',
                  'font-semibold',
                )}
                variant="outline"
                asChild
              >
                <a href="/architecture">
                  <MapIcon className="mr-2 h-5 w-5 text-gray-500" />{' '}
                  Architecture
                </a>
              </Button>
            </div>
          )}

          <div className="flex flex-1 gap-x-4 self-stretch lg:gap-x-6">
            <div className="ml-auto flex items-center gap-x-4 lg:gap-x-6">
              {data?.currentVersion &&
              data?.latestVersion &&
              data.currentVersion < data.latestVersion ? (
                <Popover>
                  <PopoverTrigger asChild>
                    <Button
                      variant="destructive"
                      className={
                        'bg-orange-500 font-semibold transition-colors hover:bg-orange-600 data-[state="open"]:bg-orange-600'
                      }
                    >
                      <span className="hidden lg:block">Update Available</span>
                      <span className="lg:hidden">Update</span>
                      <ExclamationCircleIcon
                        className={
                          'ml-2 h-5 w-5 transition duration-150 ease-in-out group-hover:text-opacity-80'
                        }
                        aria-hidden="true"
                      />
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-screen max-w-md flex-auto overflow-hidden rounded-3xl bg-white p-0 text-sm leading-6 shadow-lg ring-1 ring-gray-900/5">
                    <div className="p-4">
                      <h3 className="font mb-2 text-center text-sm font-semibold leading-6 text-gray-500">
                        A new version of Nitric is available
                      </h3>
                      <div className="group relative flex gap-x-6 rounded-lg p-4 hover:bg-gray-50">
                        <div className="mt-1 flex h-11 w-11 flex-none items-center justify-center rounded-lg bg-gray-50 group-hover:bg-white">
                          <DocumentDuplicateIcon
                            className="h-6 w-6 text-gray-600 group-hover:text-primary"
                            aria-hidden="true"
                          />
                        </div>
                        <div>
                          <a
                            href={'https://nitric.io/docs/installation'}
                            target="_blank"
                            rel="noreferrer"
                            className="font-semibold text-gray-900"
                          >
                            Upgrade from version &apos;
                            {data.currentVersion}&apos; to &apos;
                            {data.latestVersion}&apos;
                            <span className="absolute inset-0" />
                          </a>
                          <p className="mt-1 text-gray-600">
                            To upgrade, visit the installation docs for
                            instructions and release notes
                          </p>
                        </div>
                      </div>
                    </div>

                    <div className="bg-gray-50">
                      <div className="flex flex-col justify-between">
                        <h3 className="font p-4 text-center text-sm font-semibold leading-6 text-gray-500">
                          Reach out to the community
                        </h3>
                        <div className="grid grid-cols-2 divide-x divide-gray-900/5">
                          {communityLinks
                            .filter((item) => item.name !== 'Sponsor')
                            .map((item) => (
                              <a
                                key={item.name}
                                href={item.href}
                                target="_blank"
                                rel="noreferrer"
                                className="flex items-center justify-center gap-x-2.5 p-3 font-semibold text-gray-900 hover:bg-gray-100"
                              >
                                <item.icon
                                  className="h-5 w-5 flex-none text-gray-400"
                                  aria-hidden="true"
                                />
                                {item.name}
                              </a>
                            ))}
                        </div>
                      </div>
                    </div>
                  </PopoverContent>
                </Popover>
              ) : null}
              <span className="hidden font-semibold lg:block">
                Local Dashboard
              </span>
              <div className="relative">
                <Popover>
                  <PopoverTrigger asChild>
                    <Button variant="outline" className="group font-semibold">
                      <QuestionMarkCircleIcon
                        className={
                          'mr-2 h-5 w-5 text-gray-500 transition duration-150 ease-in-out group-hover:text-opacity-80 group-data-[state="open"]:text-opacity-90'
                        }
                        aria-hidden="true"
                      />
                      <span>Help</span>
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="mr-2 w-screen max-w-md flex-auto overflow-hidden rounded-3xl bg-white p-0 text-sm leading-6 shadow-lg ring-1 ring-gray-900/5">
                    <div className="p-4">
                      <h3 className="font mb-2 text-center text-sm font-semibold leading-6 text-gray-500">
                        Need help with your project?
                      </h3>
                      {resourceLinks.map((item) => (
                        <div
                          key={item.name}
                          className="group relative flex gap-x-6 rounded-lg p-4 hover:bg-gray-50"
                        >
                          <div className="mt-1 flex h-11 w-11 flex-none items-center justify-center rounded-lg bg-gray-50 group-hover:bg-white">
                            <item.icon
                              className="h-6 w-6 text-gray-600 group-hover:text-primary"
                              aria-hidden="true"
                            />
                          </div>
                          <div>
                            <a
                              href={item.href}
                              target="_blank"
                              rel="noreferrer"
                              className="font-semibold text-gray-900"
                            >
                              {item.name}
                              <span className="absolute inset-0" />
                            </a>
                            <p className="mt-1 text-gray-600">
                              {item.description}
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>

                    <div className="bg-gray-50">
                      <div className="flex flex-col justify-between">
                        <h3 className="font p-4 text-center text-sm font-semibold leading-6 text-gray-500">
                          Reach out to the community
                        </h3>
                        <div className="grid grid-cols-2 divide-x divide-gray-900/5">
                          {communityLinks
                            .filter((item) => item.name !== 'Sponsor')
                            .map((item) => (
                              <a
                                key={item.name}
                                href={item.href}
                                target="_blank"
                                rel="noreferrer"
                                className="flex items-center justify-center gap-x-2.5 p-3 font-semibold text-gray-900 hover:bg-gray-100"
                              >
                                <item.icon
                                  className="h-5 w-5 flex-none text-gray-400"
                                  aria-hidden="true"
                                />
                                {item.name}
                              </a>
                            ))}
                        </div>
                        <p className="ml-auto w-full truncate border-t px-4 py-2 text-center text-gray-400">
                          CLI Version: v{data?.currentVersion}
                        </p>
                      </div>
                    </div>
                  </PopoverContent>
                </Popover>
              </div>
            </div>
          </div>
        </div>
        {showAlert && (
          <Alert className="flex flex-col items-center justify-center rounded-none bg-primary/90 text-white">
            <AlertTitle className="flex items-center justify-center gap-4 text-xl text-white">
              Waiting for your application to start
              <Spinner color="info" className="mb-0.5" />
            </AlertTitle>
            <AlertDescription className="text-center text-lg">
              {!data
                ? 'Dashboard disconnected from nitric server, ensure nitric is running by executing `nitric start`.'
                : "Nitric is running but hasn't received a connection from your application, ensure your application is running."}
            </AlertDescription>
          </Alert>
        )}
        <div className={secondLevelNav ? 'lg:pl-80' : undefined}>
          <div className={cn('px-4 py-8 sm:px-6 lg:px-8', mainClassName)}>
            <h1
              className={cn('mb-12 text-4xl font-bold', hideTitle && 'sr-only')}
            >
              {title}
            </h1>
            {children}
          </div>
        </div>
      </main>
    </TooltipProvider>
  )
}

export default AppLayout
