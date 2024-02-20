import { type ComponentType } from 'react'

import type { Topic } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type TopicNodeData = NodeBaseData<Topic>

export const TopicNode: ComponentType<NodeProps<TopicNodeData>> = (props) => {
  const { data } = props
  //http://localhost:4001/topics/updates
  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Topic - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'topic',
        testHref: `/topics`, // TODO add url param to switch to resource
        address: `http://${data.address}`,
        services: data.resource.requestingServices,
      }}
    />
  )
}
