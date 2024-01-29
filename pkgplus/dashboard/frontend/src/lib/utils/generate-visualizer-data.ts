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
import { title } from "radash";

import {
  OpenAPIV3
} from 'openapi-types';

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
  policies: Policy[],
  data: T
): { node: Node<T>; edges: Edge[] } => {
  const edges: Edge[] = [];
  const nodeId = `${type}-${resource.name}`;

  // Generate edges from requestingServices
  resource.requestingServices.forEach((service) => {
    let edgeLabel = "";
    let source = nodeId;
    let target = service;

    const policy = policies.find((p) =>
      p.resources.some((r) => r.name === resource.name) && p.principals.some((principal) => principal.name === service)
    );

    console.log(resource.name, policy);

    if (policy) {
      source = service;
      target = nodeId;
      edgeLabel = policy?.actions
        .map((action) => title(action).split(" ").pop())
        .join(", ");
    } else if (type === "api") {
      edgeLabel = "Routes";
    } else if (type === "schedule") {
      edgeLabel = "Triggers";
    } else if (type === "topic") {
      edgeLabel = "Subscribes"
    } else if (type === 'bucket') {
      edgeLabel = "Notifies"
    }

    const edge: Edge = {
      id: `e-${source}-${target}`,
      source,
      target,
      data: {
        label: edgeLabel,
      },
    };
    edges.push(edge);
  });

  return {
    node: {
      id: nodeId,
      position: { x: 0, y: 0 },
      type,
      data,
    },
    edges,
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

    const { node, edges: apiEdges } = createNode<ApiNodeData>(
      api,
      "api",
      policies,
      {
        title: api.name,
        resource: api,
        icon: GlobeAltIcon,
        description: `${routes.length} ${
          routes.length === 1 ? "Route" : "Routes"
        }`,
      }
    );

    const specEntries = (api.spec && api.spec.paths) || [];

    Object.entries(specEntries).forEach(([path, operations]) => {
      AllHttpMethods.forEach((m) => {
        const method = operations && operations[m] as any

        if (!method) {
          return;
        }

        edges.push({
          id: `e-${api.name}-${path}-${method}`,
          source: `api-${api.name}`,
          target: method['x-nitric-target']['name'],
          data: {
            label: "Routes",
          },
        })
      })
    });

    nodes.push(node);
    // edges.push(...apiEdges);
  });

  // Generate nodes from websockets
  data.websockets.forEach((ws) => {
    const { node, edges: wsEdges } = createNode<WebsocketNodeData>(
      ws,
      "websocket",
      policies,
      {
        title: ws.name,
        resource: ws,
        icon: ChatBubbleLeftRightIcon,
        description: `${ws.events.length} ${
          ws.events.length === 1 ? "Event" : "Events"
        }`,
      }
    );

    edges.push(...Object.entries(ws.targets).map(([eventType, target]) => {
      return {
        id: `e-${ws.name}-${target}`,
        source: ws.name,
        target,
        data: {
          label: eventType,
        },
      };
    }));

    nodes.push(node);
  });

  // Generate nodes from schedules
  data.schedules.forEach((schedule) => {
    const { node, edges: schedulesEdges } = createNode<ScheduleNodeData>(
      schedule,
      "schedule",
      policies,
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
      policies,
      {
        title: bucket.name,
        resource: bucket,
        icon: CircleStackIcon,
        description: `${bucket.notificationCount} ${
          bucket.notificationCount === 1 ? "Notification" : "Notifications"
        }`,
      }
    );

    edges.push(...Object.keys(bucket.notifiers).map((subscriber) => {
      return {
        id: `e-${bucket.name}-${subscriber}`,
        source: `bucket-${bucket.name}`,
        target: subscriber,
      };
    }));

    nodes.push(node);
  });

  // Generate nodes from buckets
  data.topics.forEach((topic) => {
    const { node, edges: topicEdges } = createNode<TopicNodeData>(
      topic,
      "topic",
      policies,
      {
        title: topic.name,
        resource: topic,
        icon: MegaphoneIcon,
        description: ``,
      }
    );
    nodes.push(node);

    edges.push(...Object.keys(topic.subscribers).map((subscriber) => {
      return {
        id: `e-${topic.name}-${subscriber}`,
        source: `topic-${topic.name}`,
        target: subscriber,
      };
    }));

    edges.push(...Object.entries(data.policies).map(([_, policy]) => {
      return {
        id: `e-${policy.name}`,
        source: policy.principals[0].name,
        target: `${policy.resources[0].type}-${policy.resources[0].name}`,
      }
    }));
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

  data.services.forEach((service) => {
    const node: Node<ServiceNodeData> = {
      id: service.name,
      position: { x: 0, y: 0 },
      data: {
        title: `${service.name}`,
        description: "",
        resource: {},
        icon: CubeIcon,
      },
      type: "service",
    };
    nodes.push(node);
  })

  console.log("nodes:", nodes);
  console.log("edges:", edges);

  return { nodes, edges };
}
