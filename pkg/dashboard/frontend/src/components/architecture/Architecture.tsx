import ReactFlow, {
  Background,
  MiniMap,
  addEdge,
  useEdgesState,
  useNodesState,
  BackgroundVariant,
  type Node,
  useReactFlow,
  type Edge,
  ReactFlowProvider,
  Position,
  Panel,
} from 'reactflow'
import Dagre from '@dagrejs/dagre'
import 'reactflow/dist/style.css'
import './styles.css'

import AppLayout from '../layout/AppLayout'
import { useCallback, useEffect, useState } from 'react'
import { useWebSocket } from '@/lib/hooks/use-web-socket'
import ExportButton from './ExportButton'
import {
  generateArchitectureData,
  nodeTypes,
} from '@/lib/utils/generate-architecture-data'
import NitricEdge from './NitricEdge'
import { Switch } from '../ui/switch'
import { Label } from '../ui/label'

const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}))

const nodeWidth = 200
const nodeHeight = 150

const getLayoutedElements = (
  nodes: Node<any, string | undefined>[],
  edges: Edge[],
  direction = 'LR',
) => {
  const isHorizontal = direction === 'LR'
  g.setGraph({ rankdir: direction })

  edges.forEach((edge) => g.setEdge(edge.source, edge.target))
  nodes.forEach((node) =>
    g.setNode(node.id, {
      width: isHorizontal ? nodeWidth * 1.25 : nodeWidth,
      height: nodeHeight,
    }),
  )

  Dagre.layout(g)

  return {
    nodes: nodes.map((node) => {
      const { x, y } = g.node(node.id)

      return {
        ...node,
        position: {
          x: x - nodeWidth / 2,
          y: y - nodeHeight / 2,
        },
        targetPosition: isHorizontal ? Position.Left : Position.Top,
        sourcePosition: isHorizontal ? Position.Right : Position.Bottom,
      }
    }),
    edges,
  }
}

const edgeTypes = {
  nitric: NitricEdge,
}

const LOCAL_STORAGE_KEY = 'nitric-local-dash-arch-options'

interface ArchOptions {
  isHorizontal: boolean
}

const defaultOptions: ArchOptions = { isHorizontal: false }

const getOptions = (): ArchOptions => {
  try {
    const key = localStorage.getItem(LOCAL_STORAGE_KEY)

    return key ? JSON.parse(key) : defaultOptions
  } catch (e) {
    return defaultOptions
  }
}

const setOptions = (options: ArchOptions) => {
  localStorage.setItem(LOCAL_STORAGE_KEY, JSON.stringify(options))
}

function ReactFlowLayout() {
  const [isHorizontal, setIsHorizontal] = useState(getOptions().isHorizontal)
  const { fitView } = useReactFlow()
  const { data } = useWebSocket()
  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])

  const onConnect = useCallback(
    (params: any) => setEdges((eds) => addEdge(params, eds)),
    [setEdges],
  )

  useEffect(() => {
    if (!data) return

    const { nodes, edges } = generateArchitectureData(data)

    const layouted = getLayoutedElements(
      nodes,
      edges,
      isHorizontal ? 'LR' : 'TB',
    )

    setNodes([...layouted.nodes])
    setEdges([...layouted.edges])

    setOptions({ isHorizontal })

    window.requestAnimationFrame(() => {
      setTimeout(
        () =>
          fitView({
            minZoom: 1,
            maxZoom: 1.5,
            duration: 500, // animation duration of repositioning the arch diagram
          }),
        100, // ensure the diagram is 100% ready before re-fitting
      )
    })
  }, [data, isHorizontal])

  return (
    <AppLayout
      title="Architecture"
      hideTitle
      mainClassName="py-0 px-0 sm:px-0 lg:px-0 lg:py-0"
      routePath={'/architecture'}
    >
      <div className="h-full overflow-hidden">
        <div className="h-[calc(100vh-58px)] w-full overflow-x-hidden">
          <ReactFlow
            nodes={nodes}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            defaultEdgeOptions={{
              type: 'nitric',
            }}
            onConnect={onConnect}
            fitView
            fitViewOptions={{
              maxZoom: 1.5,
              minZoom: 1,
            }}
          >
            <MiniMap pannable zoomable className="!bg-blue-300" />
            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
            {data?.projectName && (
              <Panel position="top-right">
                <div className="flex items-center gap-x-6">
                  <div className="flex items-center gap-x-2">
                    <Switch
                      id="horizontal-mode"
                      aria-label="Toggle Horizontal Mode"
                      checked={isHorizontal}
                      onCheckedChange={setIsHorizontal}
                    />
                    <Label htmlFor="horizontal-mode">Horizontal</Label>
                  </div>
                  <ExportButton projectName={data.projectName} />
                </div>
              </Panel>
            )}
            <Panel position="bottom-left" className="flex flex-col gap-y-1">
              <div className="rounded-md border bg-white p-2">
                <div className="mb-2 text-center text-xs font-semibold">
                  Connector Types
                </div>
                <div className="grid grid-cols-2 items-center gap-x-4 gap-y-2 text-xs font-semibold">
                  <span className="h-1 border-b-2 border-dashed border-black" />
                  <span>Triggers</span>
                  <span className="h-1 border-b-2 border-black" />
                  <span>Dependencies</span>
                </div>
              </div>
            </Panel>
          </ReactFlow>
        </div>
      </div>
    </AppLayout>
  )
}

export default function Architecture() {
  return (
    <ReactFlowProvider>
      <ReactFlowLayout />
    </ReactFlowProvider>
  )
}
