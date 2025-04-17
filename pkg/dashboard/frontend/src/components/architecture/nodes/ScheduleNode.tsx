import { type ComponentType } from 'react'
import cronstrue from 'cronstrue'
import type { Schedule } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'

export type ScheduleNodeData = NodeBaseData<Schedule>

export const ScheduleNode: ComponentType<NodeProps<ScheduleNodeData>> = (
  props,
) => {
  const { data } = props

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Schedule - ${data.title}`,
        description: data.description,
        icon: data.icon,
        nodeType: 'schedule',
        testHref: `/schedules`, // TODO add url param to switch to resource
        services: data.resource.requestingServices,
        address: `http://${data.address}`,
        children: (
          <>
            {data.resource.expression ? (
              <>
                <div className="flex flex-col">
                  <span className="font-bold text-foreground">Cron:</span>
                  <span className="text-foreground">{data.resource.expression}</span>
                </div>
                <div className="flex flex-col">
                  <span className="font-bold text-foreground">Description:</span>
                  <span className="text-foreground">
                    {cronstrue.toString(data.resource.expression, {
                      verbose: true,
                    })}
                  </span>
                </div>
              </>
            ) : (
              <div className="flex flex-col">
                <span className="font-bold text-foreground">Rate:</span>
                <span className="text-foreground">Every {data.resource.rate}</span>
              </div>
            )}
          </>
        ),
      }}
    />
  )
}
