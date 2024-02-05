import { cn } from '@/lib/utils'
import {
  type EdgeProps,
  EdgeLabelRenderer,
  BaseEdge,
  getBezierPath,
} from 'reactflow'

export default function NitricEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  label,
  sourcePosition,
  targetPosition,
  style = {},
  markerEnd,
  selected,
  data,
}: EdgeProps) {
  const xEqual = sourceX === targetX
  const yEqual = sourceY === targetY

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX: xEqual ? sourceX + 0.0001 : sourceX,
    sourceY: yEqual ? sourceY + 0.0001 : sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  })

  return (
    <>
      <BaseEdge id={id} path={edgePath} style={style} markerEnd={markerEnd} />
      {label && (
        <EdgeLabelRenderer>
          <div
            className={cn(
              'nodrag absolute rounded-sm border bg-white p-1.5 text-[10px] font-semibold tracking-normal transition-all',
              selected ? 'border-primary' : 'border-gray-500',
            )}
            style={{
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            }}
          >
            {label}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}
