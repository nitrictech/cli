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
        children: (
          <div className="space-y-4">
            {data.resource.expression ? (
              <>
                <div className="flex flex-col">
                  <span className="font-bold">Cron:</span>
                  <span>{data.resource.expression}</span>
                </div>
                <div className="flex flex-col">
                  <span className="font-bold">Description:</span>
                  <span>
                    {cronstrue.toString(data.resource.expression, {
                      verbose: true,
                    })}
                  </span>
                </div>
              </>
            ) : (
              <div className="flex flex-col">
                <span className="font-bold">Rate:</span>
                <span>Every {data.resource.rate}</span>
              </div>
            )}

            <div className="flex flex-col">
              <span className="font-bold">Requested by:</span>
              <span>{data.resource.requestingServices.join(', ')}</span>
            </div>
          </div>
        ),
      }}
    />
  )
}
