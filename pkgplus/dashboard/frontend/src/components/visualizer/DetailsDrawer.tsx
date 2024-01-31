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
export interface DetailsDrawerProps extends PropsWithChildren {
  title: string
  description?: string
  open: boolean
  testHref?: string
  footerChildren?: React.ReactNode
}

export const DetailsDrawer = ({
  title,
  description,
  children,
  footerChildren,
  open,
  testHref,
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
      <DrawerContent className="fixed inset-auto bottom-0 right-0 mt-24 flex h-full w-[400px] flex-col rounded-l-[10px] rounded-r-none bg-white">
        <div className="mx-auto w-full max-w-sm p-4">
          <DrawerHeader>
            <DrawerTitle>{title}</DrawerTitle>
            {description && (
              <DrawerDescription>{description}</DrawerDescription>
            )}
          </DrawerHeader>
          <div className="p-4">{children}</div>
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
