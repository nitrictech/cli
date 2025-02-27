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
import { DetailsDrawer } from './DetailsDrawer'
import type { Endpoint } from '@/types'
import type { ApiNodeData } from './nodes/APINode'
import type { ServiceNodeData } from './nodes/ServiceNode'
import { Button } from '../ui/button'
import APIRoutesList from '../apis/APIRoutesList'

export default function NitricEdge({
  id,
  source,
  target,
  sourceX,
  sourceY,
  targetX,
  targetY,
  label,
  // sourcePosition,
  // targetPosition,
  style = {},
  markerEnd,
  selected,
  data,
}: EdgeProps<{
  type: string
  endpoints: Endpoint[]
  apiAddress: string
}>) {
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

  const isAPIEdge = data?.type === 'api'

  const highlightEdge = selected || sourceNode?.selected || targetNode?.selected

  const Icon = (targetNode?.data as ServiceNodeData).icon

  return (
    <>
      <BaseEdge
        id={id}
        path={edgePath}
        style={{
          ...style,
          stroke: highlightEdge ? 'rgb(var(--primary))' : style.stroke,
        }}
        markerEnd={markerEnd}
      />
      {label && (
        <EdgeLabelRenderer>
          <div
            data-testid={`edge-label-${id}`}
            className={cn(
              'nodrag absolute rounded-sm border bg-background text-foreground p-1 text-[9px] font-semibold tracking-normal transition-all',
              selected ? 'border-primary' : 'border-border',
            )}
            style={{
              transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px)`,
            }}
          >
            {label.toString().toLocaleLowerCase()}
            {isAPIEdge && (
              <DetailsDrawer
                title="Routes"
                nodeType="api"
                edgeId={id}
                type="edge"
                icon={(sourceNode?.data as ApiNodeData).icon}
                open={Boolean(selected)}
                footerChildren={
                  <Button asChild>
                    <a
                      href={`vscode://file/${(targetNode?.data as ServiceNodeData).resource.filePath}`}
                    >
                      <Icon className="mr-2 h-4 w-4" />
                      <span>Open in VSCode</span>
                    </a>
                  </Button>
                }
              >
                <div className="mb-4 text-sm">
                  <span className="font-semibold">
                    {(sourceNode?.data as ApiNodeData).title}
                  </span>{' '}
                  has{' '}
                  <span className="font-semibold">
                    {data.endpoints.length}{' '}
                    {data.endpoints.length === 1 ? 'route' : 'routes'}
                  </span>{' '}
                  referenced by{' '}
                  <span className="font-semibold">
                    {(targetNode?.data as ServiceNodeData).title}
                  </span>
                </div>
                <APIRoutesList
                  apiAddress={data.apiAddress}
                  endpoints={data.endpoints}
                />
              </DetailsDrawer>
            )}
          </div>
        </EdgeLabelRenderer>
      )}
    </>
  )
}
