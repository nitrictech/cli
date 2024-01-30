import {
  APINode,
  type ApiNodeData,
} from "@/components/visualizer/nodes/APINode";
import {
  BucketNode,
  type BucketNodeData,
} from "@/components/visualizer/nodes/BucketNode";
import type { BaseResource, Policy, WebSocketResponse } from "@/types";
import {
  ChatBubbleLeftRightIcon,
  CircleStackIcon,
  ClockIcon,
  CubeIcon,
  MegaphoneIcon,
  GlobeAltIcon,
} from "@heroicons/react/24/outline";
import { MarkerType, type Edge, type Node } from "reactflow";
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

import { OpenAPIV3 } from "openapi-types";

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
): Node<T> => {
  const nodeId = `${type}-${resource.name}`;

  // Generate edges from requestingServices
  return {
    id: nodeId,
    position: { x: 0, y: 0 },
    type,
    data,
  };
};

const AllHttpMethods = [
  OpenAPIV3.HttpMethods.GET,
  OpenAPIV3.HttpMethods.PUT,
  OpenAPIV3.HttpMethods.POST,
  OpenAPIV3.HttpMethods.DELETE,
  OpenAPIV3.HttpMethods.OPTIONS,
  // OpenAPIV3.HttpMethods.HEAD,
  // OpenAPIV3.HttpMethods.PATCH,
  // OpenAPIV3.HttpMethods.TRACE,
];

const actionVerbs = [
  "Get",
  "List",
  "Put",
  "Delete",
  "Publish",
  "Detail",
];

function verbFromNitricAction(action: string) {
  for (const verb of actionVerbs) {
    if (action.endsWith(verb)) {
      return verb;
    }
  }

  return action;
}

export function generateVisualizerData(data: WebSocketResponse): {
  nodes: Node[];
  edges: Edge[];
} {
  const nodes: Node[] = [];
  const edges: Edge[] = [];
  // const uniqueServices: Set<string> = new Set();
  const policies = Object.entries(data.policies).map(([_, p]) => p);

  console.log("data:", data);
  // console.log("apis:", data.apis);

  // Generate nodes from APIs
  data.apis.forEach((api) => {
    const routes = (api.spec && Object.keys(api.spec.paths)) || [];

    const node = createNode<ApiNodeData>(api, "api", {
      title: api.name,
      resource: api,
      icon: GlobeAltIcon,
      description: `${routes.length} ${
        routes.length === 1 ? "Route" : "Routes"
      }`,
    });

    const specEntries = (api.spec && api.spec.paths) || [];

    Object.entries(specEntries).forEach(([path, operations]) => {
      AllHttpMethods.forEach((m) => {
        const method = operations && (operations[m] as any);

        if (!method) {
          return;
        }

        edges.push({
          id: `e-${api.name}-${path}-${m}`,
          source: `api-${api.name}`,
          target: method["x-nitric-target"]["name"],
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: "auto-start-reverse",
          },
          label: `${m} ${path}`,
        });
      });
    });

    nodes.push(node);
  });

  // Generate nodes from websockets
  data.websockets.forEach((ws) => {
    const node = createNode<WebsocketNodeData>(ws, "websocket", {
      title: ws.name,
      resource: ws,
      icon: ChatBubbleLeftRightIcon,
      description: `${ws.events.length} ${
        ws.events.length === 1 ? "Event" : "Events"
      }`,
    });

    edges.push(
      ...Object.entries(ws.targets).map(([eventType, target]) => {
        return {
          id: `e-${ws.name}-${target}`,
          source: ws.name,
          target,
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: "auto-start-reverse",
          },
          label: eventType,
        };
      })
    );

    nodes.push(node);
  });

  // Generate nodes from schedules
  data.schedules.forEach((schedule) => {
    const node = createNode<ScheduleNodeData>(schedule, "schedule", {
      title: schedule.name,
      resource: schedule,
      icon: ClockIcon,
      description: ``,
    });

    nodes.push(node);

    edges.push({
        id: `e-${schedule.name}-${schedule.target}`,
        source: `schedule-${schedule.name}`,
        target: schedule.target,
        animated: true,
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
        markerStart: {
          type: MarkerType.ArrowClosed,
          orient: 'auto-start-reverse',
        },
        label: "Triggers",
      });
  });

  // Generate nodes from buckets
  data.buckets.forEach((bucket) => {
    const node = createNode<BucketNodeData>(bucket, "bucket", {
      title: bucket.name,
      resource: bucket,
      icon: CircleStackIcon,
      description: `${bucket.notificationCount} ${
        bucket.notificationCount === 1 ? "Notification" : "Notifications"
      }`,
    });

    edges.push(
      ...Object.keys(bucket.notifiers).map((subscriber) => {
        return {
          id: `e-${bucket.name}-${subscriber}`,
          source: `bucket-${bucket.name}`,
          target: subscriber,
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: "auto-start-reverse",
          },
          label: "Notifies",
        };
      })
    );

    nodes.push(node);
  });

  // Generate nodes from buckets
  data.topics.forEach((topic) => {
    const node = createNode<TopicNodeData>(topic, "topic", {
      title: topic.name,
      resource: topic,
      icon: MegaphoneIcon,
      description: ``,
    });
    nodes.push(node);

    edges.push(
      ...Object.keys(topic.subscribers).map((subscriber) => {
        return {
          id: `e-${topic.name}-${subscriber}`,
          source: `topic-${topic.name}`,
          target: subscriber,
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: "auto-start-reverse",
          },
          label: "Subscribes",
        };
      })
    );
  });

  edges.push(
    ...Object.entries(data.policies).map(([_, policy]) => {
      return {
        id: `e-${policy.name}`,
        source: policy.principals[0].name,
        target: `${policy.resources[0].type}-${policy.resources[0].name}`,
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
        markerStart: {
          type: MarkerType.ArrowClosed,
          orient: "auto-start-reverse",
        },
        label: policy.actions.map(verbFromNitricAction).join(", "),
      } as Edge;
    })
  );

  data.services.forEach((service) => {
    const node: Node<ServiceNodeData> = {
      id: service.name,
      position: { x: 0, y: 0 },
      data: {
        title: `${service.name.replace(/\\/g, "/")}`,
        description: "",
        resource: {},
        icon: CubeIcon,
      },
      type: "service",
    };
    nodes.push(node);
  });

  console.log("nodes:", nodes);
  console.log("edges:", edges);

  return { nodes, edges };
}
