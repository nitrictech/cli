import { type ComponentType } from 'react'

import type { Edge, NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import { Button } from '@/components/ui/button'

type BatchData = {
  filePath: string
}

export interface BatchNodeData extends NodeBaseData<BatchData> {
  connectedEdges: Edge[]
}

export const BatchNode: ComponentType<NodeProps<BatchNodeData>> = (props) => {
  const { data } = props

  const Icon = data.icon

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Batch - ${data.title}`,
        icon: Icon,
        nodeType: 'batch',
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
