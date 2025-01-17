import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarGroupAction,
  SidebarGroupContent,
  SidebarGroupLabel,
} from '@/components/ui/sidebar'
import { Button } from '../ui/button'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '../ui/collapsible'
import { ChevronRightIcon } from '@heroicons/react/24/outline'
import { useMemo, type PropsWithChildren } from 'react'
import { MultiSelect } from '../shared/MultiSelect'
import { useParams } from '@/hooks/use-params'
import { useWebSocket } from '@/lib/hooks/use-web-socket'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select'

interface CollapsibleGroupProps extends PropsWithChildren {
  title: string
  defaultOpen?: boolean
}

const CollapsibleGroup = ({
  title,
  children,
  defaultOpen,
}: CollapsibleGroupProps) => {
  return (
    <Collapsible defaultOpen={defaultOpen} className="group/collapsible">
      <SidebarGroup className="p-0">
        <SidebarGroupLabel
          asChild
          className="flex h-full w-full items-center p-2 text-base font-semibold text-foreground text-gray-600 hover:bg-gray-100 group-data-[state=open]/collapsible:text-foreground"
        >
          <CollapsibleTrigger
            data-testid={`filter-${title.toLowerCase().replace(' ', '-')}-collapsible`}
          >
            <ChevronRightIcon className="mr-2 !size-6 transition-transform group-data-[state=open]/collapsible:rotate-90" />
            <span className="mb-0.5">{title}</span>
          </CollapsibleTrigger>
        </SidebarGroupLabel>
        <CollapsibleContent>
          <SidebarGroupContent className="p-2">{children}</SidebarGroupContent>
        </CollapsibleContent>
      </SidebarGroup>
    </Collapsible>
  )
}

const levelsList = [
  { value: 'error', label: 'Error' },
  { value: 'warning', label: 'Warning' },
  { value: 'info', label: 'Info' },
]

export function FilterSidebar() {
  const { data } = useWebSocket()
  const { searchParams, setParams } = useParams()

  const levels =
    searchParams
      .get('level')
      ?.split(',')
      .filter((l) => levelsList.some((o) => o.value === l)) ?? []
  const origins =
    searchParams
      .get('origin')
      ?.split(',')
      .filter((o) => {
        return (
          o === 'nitric' ||
          data?.services.some((service) => service.name === o) ||
          data?.batchServices.some((service) => service.name === o)
        )
      }) ?? []
  const timeline = searchParams.get('timeline')

  const originsList = useMemo(() => {
    return [
      {
        label: 'nitric',
        value: 'nitric',
      },
      ...(data?.services.map((service) => ({
        value: service.name,
        label: service.name,
      })) ?? []),
      ...(data?.batchServices.map((service) => ({
        value: service.name,
        label: service.name,
      })) ?? []),
    ]
  }, [data?.services])

  const handleResetFilters = () => {
    setParams('level', null)
    setParams('origin', null)
    setParams('timeline', null)
  }

  return (
    <Sidebar className="ml-20 mt-16">
      <SidebarContent className="gap-0 pt-7">
        <SidebarGroup>
          <SidebarGroupLabel className="text-base font-semibold text-foreground">
            Filters
          </SidebarGroupLabel>
          <SidebarGroupAction title="Reset Filters" asChild>
            <Button
              variant="outline"
              data-testid="filter-logs-reset-btn"
              size="sm"
              onClick={handleResetFilters}
              className="absolute right-2 top-2 h-8 w-16 font-semibold text-foreground"
            >
              Reset
            </Button>
          </SidebarGroupAction>
          <SidebarGroupContent className="pt-4">
            <CollapsibleGroup title="Timeline">
              <Select
                value={timeline ?? ''}
                onValueChange={(v) => {
                  setParams('timeline', v)
                }}
              >
                <SelectTrigger
                  className="w-full"
                  data-testid="timeline-select-trigger"
                >
                  <SelectValue placeholder="select" />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    <SelectItem value={''}>Maximum (all logs)</SelectItem>
                    <SelectItem value={'pastHalfHour'}>
                      Past 30 minutes
                    </SelectItem>
                    <SelectItem value={'pastHour'}>Past Hour</SelectItem>
                    <SelectItem value={'pastDay'}>Past 24 Hours</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </CollapsibleGroup>
            <CollapsibleGroup title="Contains Level">
              <MultiSelect
                options={levelsList}
                onValueChange={(value) => setParams('level', value.join(','))}
                value={levels}
                placeholder="Select severity levels"
                variant="inverted"
                disableSelectAll
                data-testid="level-select"
                maxCount={3}
              />
            </CollapsibleGroup>
            <CollapsibleGroup title="Origin">
              <MultiSelect
                options={originsList}
                onValueChange={(value) => setParams('origin', value.join(','))}
                value={origins}
                placeholder="Select origins"
                variant="inverted"
                data-testid="origin-select"
                disableSelectAll
                maxCount={3}
              />
            </CollapsibleGroup>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
