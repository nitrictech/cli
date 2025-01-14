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
import { ChevronDownIcon } from '@heroicons/react/24/outline'

export function FilterSidebar() {
  return (
    <Sidebar className="ml-20 mt-16">
      <SidebarContent className="gap-0 pt-7">
        <SidebarGroup>
          <SidebarGroupLabel className="text-base font-semibold text-foreground">
            Filters
          </SidebarGroupLabel>
          <SidebarGroupAction title="Reset Filters" className="mr-6">
            <Button variant="outline" className="font-semibold text-foreground">
              Reset
            </Button>
          </SidebarGroupAction>
          <SidebarGroupContent>
            <Collapsible defaultOpen className="group/collapsible">
              <SidebarGroup>
                <SidebarGroupLabel
                  asChild
                  className="text-base font-semibold text-foreground"
                >
                  <CollapsibleTrigger>
                    Origin
                    <ChevronDownIcon className="ml-auto h-6 w-6 transition-transform group-data-[state=open]/collapsible:rotate-180" />
                  </CollapsibleTrigger>
                </SidebarGroupLabel>
                <CollapsibleContent>
                  <SidebarGroupContent>s</SidebarGroupContent>
                </CollapsibleContent>
              </SidebarGroup>
            </Collapsible>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
    </Sidebar>
  )
}
