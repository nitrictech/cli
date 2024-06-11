import { type ComponentType } from 'react'

import type { SQLDatabase } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import React from 'react'

export type SQLNodeData = NodeBaseData<SQLDatabase>

export const SQLNode: ComponentType<NodeProps<SQLNodeData>> = (props) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `SQL Database - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'sql',
        services: data.resource.requestingServices,
      }}
    />
  )
}
