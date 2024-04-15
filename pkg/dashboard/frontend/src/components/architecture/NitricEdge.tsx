import { cn } from '@/lib/utils'
import { getEdgeParams } from '@/lib/utils/generate-architecture-data'
import {
  type EdgeProps,
  EdgeLabelRenderer,
  BaseEdge,
  getBezierPath,
  useNodes,
  useStore,
  type ReactFlowState,
} from 'reactflow'

export default function NitricEdge({
  id,
  source,
  target,
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
  const allNodes = useNodes()

  const xEqual = sourceX === targetX
  const yEqual = sourceY === targetY

  const isBiDirectionEdge = useStore((s: ReactFlowState) => {
    const edgeExists = s.edges.some(
      (e) =>
        (e.source === target && e.target === source) ||
        (e.target === source && e.source === target),
    )

    return edgeExists
  })

  const sourceNode = allNodes.find((n) => n.id === source)
  const targetNode = allNodes.find((n) => n.id === target)

  const edgeParams = getEdgeParams(sourceNode, targetNode)

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX: edgeParams.sx,
    sourceY: edgeParams.sy,
    sourcePosition: isBiDirectionEdge
      ? edgeParams.targetPos
      : edgeParams.sourcePos,
    targetX: edgeParams.tx,
    targetY: edgeParams.ty,
    targetPosition: edgeParams.targetPos,
    curvature: isBiDirectionEdge ? -0.05 : undefined,
  })

  return (
    <>
      <BaseEdge id={id} path={edgePath} style={style} markerEnd={markerEnd} />
      {label && (
        <EdgeLabelRenderer>
          <div
            className={cn(
              'nodrag absolute rounded-sm border bg-white p-1 text-[9px] font-semibold tracking-normal transition-all',
              selected ? 'border-primary' : 'border-gray-500',
            )}
            style={{
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            }}
          >
            {label.toString().toLocaleLowerCase()}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}
