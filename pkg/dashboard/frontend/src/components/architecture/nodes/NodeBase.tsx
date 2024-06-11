import type { FC, PropsWithChildren } from 'react'
import { Handle, Position, type NodeProps } from 'reactflow'
import { DetailsDrawer, type DetailsDrawerProps } from '../DetailsDrawer'
import type { IconType } from 'react-icons/lib'

export interface NodeBaseData<T = Record<string, any>> {
  resource: T
  title: string
  icon:
    | React.ForwardRefExoticComponent<
        Omit<React.SVGProps<SVGSVGElement>, 'ref'> & {
          title?: string | undefined
          titleId?: string | undefined
        } & React.RefAttributes<SVGSVGElement>
      >
    | IconType
  description?: string
  address?: string
}

interface Props extends NodeProps<NodeBaseData> {
  drawerOptions?: Omit<DetailsDrawerProps, 'open'>
}

const NodeBase: FC<PropsWithChildren<Props>> = ({
  drawerOptions,
  selected,
  dragging,
  data: { icon: Icon, title },
}) => {
  return (
    <>
      {drawerOptions && (
        <DetailsDrawer {...drawerOptions} open={selected && !dragging} />
      )}
      <div className="wrapper gradient relative flex flex-grow overflow-hidden rounded-md p-0.5">
        <div className="relative flex grow items-center justify-center gap-4 rounded bg-white pr-4">
          <div className="flex h-full w-14 items-center justify-center">
            <div className="gradient relative flex size-10 items-center justify-center overflow-hidden rounded-full">
              <div className="z-10 flex size-9 items-center justify-center rounded-full bg-white">
                <Icon className={'resource-icon size-6'} />
              </div>
            </div>
          </div>
          <div className="text-sm font-semibold">{title}</div>

          <Handle type="target" isConnectable={false} position={Position.Top} />
          <Handle
            type="source"
            isConnectable={false}
            position={Position.Bottom}
          />
        </div>
      </div>
    </>
  )
}

export default NodeBase
