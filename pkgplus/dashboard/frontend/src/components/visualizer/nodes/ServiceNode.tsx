import { type ComponentType } from 'react'

import type { Edge, NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import { Button } from '@/components/ui/button'

type ServiceData = {
  filePath: string
}

export interface ServiceNodeData extends NodeBaseData<ServiceData> {
  connectedEdges: Edge[]
}

export const ServiceNode: ComponentType<NodeProps<ServiceNodeData>> = (
  props,
) => {
  const { data } = props

  const Icon = data.icon

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Service - ${data.title}`,
        icon: Icon,
        nodeType: 'service',
        description: data.description,
        footerChildren: (
          <Button asChild>
            <a href={`vscode://file/${data.resource.filePath}`}>
              <Icon className="mr-2 h-4 w-4" />
              <span>Open in VSCode</span>
            </a>
          </Button>
        ),
      }}
    />
  )
}
