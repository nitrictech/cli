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
} from "reactflow";
import Dagre from "@dagrejs/dagre";
import "reactflow/dist/style.css";
import "./styles.css";

import AppLayout from "../layout/AppLayout";
import { useCallback, useEffect } from "react";
import { useWebSocket } from "@/lib/hooks/use-web-socket";
import ExportButton from "./ExportButton";
import {
  generateVisualizerData,
  nodeTypes,
} from "@/lib/utils/generate-visualizer-data";
import NitricEdge from "./NitricEdge";

const g = new Dagre.graphlib.Graph().setDefaultEdgeLabel(() => ({}));

const nodeWidth = 150;
const nodeHeight = 150;

const getLayoutedElements = (
  nodes: Node<any, string | undefined>[],
  edges: Edge[],
  direction = "LR"
) => {
  const isHorizontal = direction === "LR";
  g.setGraph({ rankdir: direction });

  edges.forEach((edge) => g.setEdge(edge.source, edge.target));
  nodes.forEach((node) =>
    g.setNode(node.id, { width: nodeWidth, height: nodeHeight })
  );

  Dagre.layout(g);

  return {
    nodes: nodes.map((node) => {
      const { x, y } = g.node(node.id);

      return {
        ...node,
        position: {
          x: x - nodeWidth / 2,
          y: y - nodeHeight / 2,
        },
        targetPosition: isHorizontal ? Position.Left : Position.Top,
        sourcePosition: isHorizontal ? Position.Right : Position.Bottom,
      };
    }),
    edges,
  };
};

const edgeTypes = {
  nitric: NitricEdge,
};

function ReactFlowLayout() {
  const { fitView } = useReactFlow();
  const { data } = useWebSocket();
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  const onConnect = useCallback(
    (params: any) => setEdges((eds) => addEdge(params, eds)),
    [setEdges]
  );

  useEffect(() => {
    if (!data) return;

    const { nodes, edges } = generateVisualizerData(data);

    const layouted = getLayoutedElements(nodes, edges, "LB");

    setNodes([...layouted.nodes]);
    setEdges([...layouted.edges]);

    window.requestAnimationFrame(() => {
      fitView();
    });
  }, [data]);

  return (
    <AppLayout
      title="Visualizer"
      hideTitle
      mainClassName="py-0 px-0 sm:px-0 lg:px-0 lg:py-0"
      routePath={"/visualizer"}
    >
      <div className="overflow-hidden h-full">
        <div className="w-full h-[calc(100vh-58px)] overflow-x-hidden">
          <ReactFlow
            nodes={nodes}
            nodeTypes={nodeTypes}
            edgeTypes={edgeTypes}
            edges={edges}
            onNodesChange={onNodesChange}
            onEdgesChange={onEdgesChange}
            defaultEdgeOptions={{
              type: "nitric",
            }}
            onConnect={onConnect}
            fitView
          >
            <MiniMap pannable zoomable className="!bg-blue-300" />
            <Background variant={BackgroundVariant.Dots} gap={12} size={1} />
            {data?.projectName && (
              <Panel position="top-right">
                <ExportButton projectName={data.projectName} />
              </Panel>
            )}
          </ReactFlow>
        </div>
      </div>
    </AppLayout>
  );
}

export default function Visualizer() {
  return (
    <ReactFlowProvider>
      <ReactFlowLayout />
    </ReactFlowProvider>
  );
}
