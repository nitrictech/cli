import { type ComponentType } from 'react'

import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import { CodeBracketIcon } from '@heroicons/react/24/outline'
import { Button } from '@/components/ui/button'

type ServiceData = {
  filePath: string
}

export type ServiceNodeData = NodeBaseData<ServiceData>

export const ServiceNode: ComponentType<NodeProps<ServiceNodeData>> = (
  props,
) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        children: (
          <div className="flex flex-col">
            <Button asChild>
              <a href={`vscode://file/${data.resource.filePath}`}>
                <CodeBracketIcon className="mr-2 h-4 w-4" />
                <span>Open in VScode</span>
              </a>
            </Button>
          </div>
        ),
      }}
    />
  )
}
