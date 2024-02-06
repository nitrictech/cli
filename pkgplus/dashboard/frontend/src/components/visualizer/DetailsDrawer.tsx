import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '../ui/drawer'
import { Button } from '../ui/button'
import type { PropsWithChildren } from 'react'
import { useStoreApi } from 'reactflow'
import type { NodeBaseData } from './nodes/NodeBase'
import type { nodeTypes } from '@/lib/utils/generate-visualizer-data'
export interface DetailsDrawerProps extends PropsWithChildren {
  title: string
  description?: string
  open: boolean
  testHref?: string
  footerChildren?: React.ReactNode
  nodeType: keyof typeof nodeTypes
  icon: NodeBaseData['icon']
}

export const DetailsDrawer = ({
  title,
  description,
  children,
  footerChildren,
  open,
  testHref,
  icon: Icon,
  nodeType,
}: DetailsDrawerProps) => {
  const store = useStoreApi()

  return (
    <Drawer
      direction="right"
      open={open}
      onOpenChange={(open) => {
        if (!open) {
          const { unselectNodesAndEdges } = store.getState()
          unselectNodesAndEdges()
        }
      }}
    >
      <DrawerContent className="fixed inset-auto bottom-0 right-0 mt-24 flex h-full w-[380px] flex-col rounded-l-[10px] rounded-r-none bg-white">
        <div className="mx-auto w-full max-w-sm p-4">
          <DrawerHeader
            className={`flex items-center react-flow__node-${nodeType}`}
          >
            <Icon className="resource-icon mr-2 size-8" />
            <div>
              <DrawerTitle>
                <span className="flex items-center">{title}</span>
              </DrawerTitle>
              {description && (
                <DrawerDescription>{description}</DrawerDescription>
              )}
            </div>
          </DrawerHeader>
          <div className="space-y-2 p-4">{children}</div>
          <DrawerFooter>
            {footerChildren}
            {testHref && (
              <Button asChild>
                <a href={testHref} target="_blank" rel="noreferrer">
                  Test
                </a>
              </Button>
            )}
            <DrawerClose asChild>
              <Button variant="outline">Close</Button>
            </DrawerClose>
          </DrawerFooter>
        </div>
      </DrawerContent>
    </Drawer>
  )
}
