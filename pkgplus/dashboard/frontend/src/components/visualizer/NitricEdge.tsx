import { cn } from '@/lib/utils'
import { getEdgeParams } from '@/lib/utils/generate-visualizer-data';
import {
  type EdgeProps,
  EdgeLabelRenderer,
  BaseEdge,
  getBezierPath,
  useNodes,
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
  const allNodes = useNodes();

  const xEqual = sourceX === targetX
  const yEqual = sourceY === targetY

  const sourceNode = allNodes.find(n => n.id === source);
  const targetNode = allNodes.find(n => n.id === target);

  const edgeParams = getEdgeParams(sourceNode, targetNode);

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX: edgeParams.sx,
    sourceY: edgeParams.sy,
    sourcePosition: edgeParams.sourcePos,
    targetX: edgeParams.tx,
    targetY: edgeParams.ty,
    targetPosition: edgeParams.targetPos,
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
