import {
  Drawer,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from '../ui/drawer'
import { Button } from '../ui/button'
import { useCallback, type PropsWithChildren } from 'react'
import { applyNodeChanges, useNodes, useNodeId, useReactFlow } from 'reactflow'
import type { NodeBaseData } from './nodes/NodeBase'
import type { nodeTypes } from '@/lib/utils/generate-architecture-data'
export interface DetailsDrawerProps extends PropsWithChildren {
  title: string
  description?: string
  open: boolean
  testHref?: string
  footerChildren?: React.ReactNode
  nodeType: keyof typeof nodeTypes
  icon: NodeBaseData['icon']
  address?: string
  services?: string[]
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
  address,
  services,
}: DetailsDrawerProps) => {
  const nodeId = useNodeId()
  const { setNodes } = useReactFlow()
  const nodes = useNodes()

  const selectServiceNode = useCallback(
    (serviceNodeId: string) => {
      setNodes(
        applyNodeChanges(
          [
            {
              id: nodeId || '',
              type: 'select',
              selected: false,
            },
            {
              id: serviceNodeId,
              type: 'select',
              selected: true,
            },
          ],
          nodes,
        ),
      )
    },
    [nodes, nodeId],
  )

  const close = () => {
    setNodes(
      applyNodeChanges(
        [
          {
            id: nodeId || '',
            type: 'select',
            selected: false,
          },
        ],
        nodes,
      ),
    )
  }

  return (
    <Drawer modal={false} direction="right" open={open}>
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
          <div className="space-y-2 py-4">
            {children}
            {address && (
              <div className="flex flex-col">
                <span className="font-bold">Address:</span>
                {address.startsWith('http') ? (
                  <a
                    target="_blank"
                    className="hover:underline"
                    href={address}
                    rel="noreferrer"
                  >
                    {address}
                  </a>
                ) : (
                  address
                )}
              </div>
            )}
            {services?.length ? (
              <div className="flex flex-col">
                <span className="font-bold">Requested by:</span>
                <div className="flex flex-col items-start gap-y-1">
                  {services.map((s) => (
                    <button
                      key={s}
                      type="button"
                      className="hover:underline"
                      onClick={() => selectServiceNode(s)}
                    >
                      {s.replace(/\\/g, '/')}
                    </button>
                  ))}
                </div>
              </div>
            ) : null}
          </div>
          <DrawerFooter className="px-0">
            {footerChildren}
            {testHref && (
              <Button asChild>
                <a href={testHref} target="_blank" rel="noreferrer">
                  Test
                </a>
              </Button>
            )}
            <Button onClick={close} variant="outline">
              Close
            </Button>
          </DrawerFooter>
        </div>
      </DrawerContent>
    </Drawer>
  )
}
