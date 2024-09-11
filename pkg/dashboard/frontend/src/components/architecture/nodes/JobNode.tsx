import { type ComponentType } from 'react'

import type { BatchJob } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type JobNodeData = NodeBaseData<BatchJob>

export const JobNode: ComponentType<NodeProps<JobNodeData>> = (props) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Job - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'job',
        testHref: `/jobs`, // TODO add url param to switch to resource
        services: data.resource.requestingServices,
        address: `http://${data.address}`,
      }}
    />
  )
}
