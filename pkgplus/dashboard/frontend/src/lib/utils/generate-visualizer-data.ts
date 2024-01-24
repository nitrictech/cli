import {
  APINode,
  type ApiNodeData,
} from "@/components/visualizer/nodes/APINode";
import {
  BucketNode,
  type BucketNodeData,
} from "@/components/visualizer/nodes/BucketNode";
import type { BaseResource, WebSocketResponse } from "@/types";
import {
  ChatBubbleLeftRightIcon,
  CircleStackIcon,
  ClockIcon,
  CubeIcon,
  MegaphoneIcon,
} from "@heroicons/react/24/outline";
import GlobeAltIcon from "@heroicons/react/24/outline/GlobeAltIcon";
import type { Edge, Node } from "reactflow";
import {
  TopicNode,
  type TopicNodeData,
} from "@/components/visualizer/nodes/TopicNode";
import {
  WebsocketNode,
  type WebsocketNodeData,
} from "@/components/visualizer/nodes/WebsocketNode";
import {
  ScheduleNode,
  type ScheduleNodeData,
} from "@/components/visualizer/nodes/ScheduleNode";
import {
  ServiceNode,
  type ServiceNodeData,
} from "@/components/visualizer/nodes/ServiceNode";

export const nodeTypes = {
  api: APINode,
  bucket: BucketNode,
  schedule: ScheduleNode,
  topic: TopicNode,
  websocket: WebsocketNode,
  service: ServiceNode,
};

const createNode = <T>(
  resource: BaseResource,
  type: keyof typeof nodeTypes,
  data: T
): { node: Node<T>; edges: Edge[] } => {
  const edges: Edge[] = [];
  const nodeId = `${type}-${resource.name}`;

  // Generate edges from requestingServices
  resource.requestingServices.forEach((service) => {
    const edge: Edge = {
      id: `e-${nodeId}-${service}`,
      source: nodeId,
      target: service,
    };
    edges.push(edge);
  });

  return {
    node: {
      id: nodeId,
      position: { x: 0, y: 0 }, // Set your desired position
      type,
      data,
    },
    edges,
  };
};

export function generateVisualizerData(data: WebSocketResponse): {
  nodes: Node[];
  edges: Edge[];
} {
  const nodes: Node[] = [];
  const edges: Edge[] = [];
  const uniqueServices: Set<string> = new Set();

  // Generate nodes from APIs
  data.apis.forEach((api) => {
    const routes = Object.keys(api.spec.paths);

    const { node, edges: apiEdges } = createNode<ApiNodeData>(api, "api", {
      title: api.name,
      resource: api,
      icon: GlobeAltIcon,
      description: `An API with ${routes.length} ${
        routes.length === 1 ? "Route" : "Routes"
      }`,
    });

    nodes.push(node);
    edges.push(...apiEdges);
  });

  // Generate nodes from websockets
  data.websockets.forEach((ws) => {
    const { node, edges: wsEdges } = createNode<WebsocketNodeData>(
      ws,
      "websocket",
      {
        title: ws.name,
        resource: ws,
        icon: ChatBubbleLeftRightIcon,
        description: ``,
      }
    );

    nodes.push(node);
    edges.push(...wsEdges);
  });

  // Generate nodes from schedules
  data.schedules.forEach((schedule) => {
    const { node, edges: schedulesEdges } = createNode<ScheduleNodeData>(
      schedule,
      "schedule",
      {
        title: schedule.name,
        resource: schedule,
        icon: ClockIcon,
        description: ``,
      }
    );

    nodes.push(node);
    edges.push(...schedulesEdges);
  });

  // Generate nodes from buckets
  data.buckets.forEach((bucket) => {
    const { node, edges: bucketEdges } = createNode<BucketNodeData>(
      bucket,
      "bucket",
      {
        title: bucket.name,
        resource: bucket,
        icon: CircleStackIcon,
        description: ``,
      }
    );

    nodes.push(node);
    edges.push(...bucketEdges);
  });

  // Generate nodes from buckets
  data.topics.forEach((topic) => {
    const { node, edges: topicEdges } = createNode<TopicNodeData>(
      topic,
      "topic",
      {
        title: topic.name,
        resource: topic,
        icon: MegaphoneIcon,
        description: ``,
      }
    );

    nodes.push(node);
    edges.push(...topicEdges);
  });

  // Generate nodes for containers

  // Generate edges from policies
  // TODO use policies to add more info via edges or nodes
  // Object.values(data.policies).forEach((policy) => {
  //   policy.resources.forEach((resource) => {
  //     const edge: Edge = {
  //       id: `e-${resource.name}-${policy.name}`,
  //       source: resource.name,
  //       target: resource.name,
  //     };
  //     edges.push(edge);
  //   });
  // });

  // Collect unique services in a single pass
  edges.forEach(({ target: serviceName, id }) => {
    if (!uniqueServices.has(serviceName)) {
      console.log(serviceName);
      const node: Node<ServiceNodeData> = {
        id: serviceName,
        position: { x: 0, y: 0 },
        data: {
          title: `${serviceName}`,
          description: "",
          resource: {},
          icon: CubeIcon,
        },
        type: "service",
      };
      nodes.push(node);
      uniqueServices.add(serviceName);
    }
  });

  return { nodes, edges };
}
