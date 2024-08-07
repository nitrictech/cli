import {
  APINode,
  type ApiNodeData,
} from '@/components/architecture/nodes/APINode'
import {
  BucketNode,
  type BucketNodeData,
} from '@/components/architecture/nodes/BucketNode'
import type { BaseResource, WebSocketResponse, WebsocketEvent } from '@/types'
import {
  ChatBubbleLeftRightIcon,
  ArchiveBoxIcon,
  ClockIcon,
  CircleStackIcon,
  CpuChipIcon,
  MegaphoneIcon,
  GlobeAltIcon,
  ArrowsRightLeftIcon,
  QueueListIcon,
  LockClosedIcon,
} from '@heroicons/react/24/outline'
import {
  MarkerType,
  type Edge,
  type Node,
  Position,
  getConnectedEdges,
} from 'reactflow'
import {
  TopicNode,
  type TopicNodeData,
} from '@/components/architecture/nodes/TopicNode'
import {
  WebsocketNode,
  type WebsocketNodeData,
} from '@/components/architecture/nodes/WebsocketNode'
import { KeyValueNode } from '@/components/architecture/nodes/KeyValueNode'
import {
  ScheduleNode,
  type ScheduleNodeData,
} from '@/components/architecture/nodes/ScheduleNode'
import {
  ServiceNode,
  type ServiceNodeData,
} from '@/components/architecture/nodes/ServiceNode'

import { OpenAPIV3 } from 'openapi-types'
import { getBucketNotifications } from './get-bucket-notifications'
import {
  HttpProxyNode,
  type HttpProxyNodeData,
} from '@/components/architecture/nodes/HttpProxyNode'
import { getTopicSubscriptions } from './get-topic-subscriptions'
import { QueueNode } from '@/components/architecture/nodes/QueueNode'
import { SQLNode } from '@/components/architecture/nodes/SQLNode'
import { SiPostgresql } from 'react-icons/si'
import { unique } from 'radash'
import { SecretNode } from '@/components/architecture/nodes/SecretNode'

export const nodeTypes = {
  api: APINode,
  bucket: BucketNode,
  schedule: ScheduleNode,
  topic: TopicNode,
  websocket: WebsocketNode,
  service: ServiceNode,
  keyvaluestore: KeyValueNode,
  sql: SQLNode,
  httpproxy: HttpProxyNode,
  queue: QueueNode,
  secret: SecretNode,
}

const createNode = <T>(
  resource: BaseResource,
  type: keyof typeof nodeTypes,
  data: T,
): Node<T> => {
  const nodeId = `${type}-${resource.name}`

  return {
    id: nodeId,
    position: { x: 0, y: 0 },
    type,
    data,
  }
}

const AllHttpMethods = [
  OpenAPIV3.HttpMethods.GET,
  OpenAPIV3.HttpMethods.PUT,
  OpenAPIV3.HttpMethods.POST,
  OpenAPIV3.HttpMethods.DELETE,
  OpenAPIV3.HttpMethods.OPTIONS,
  // OpenAPIV3.HttpMethods.HEAD,
  // OpenAPIV3.HttpMethods.PATCH,
  // OpenAPIV3.HttpMethods.TRACE,
]

// this helper function returns the intersection point
// of the line between the center of the intersectionNode and the target node
function getNodeIntersection(intersectionNode: any, targetNode: any) {
  // https://math.stackexchange.com/questions/1724792/an-algorithm-for-finding-the-intersection-point-between-a-center-of-vision-and-a
  const {
    width: intersectionNodeWidth,
    height: intersectionNodeHeight,
    positionAbsolute: intersectionNodePosition,
  } = intersectionNode
  const targetPosition = targetNode.positionAbsolute

  const w = intersectionNodeWidth / 2
  const h = intersectionNodeHeight / 2

  const x2 = intersectionNodePosition.x + w
  const y2 = intersectionNodePosition.y + h
  const x1 = targetPosition.x + targetNode.width / 2
  const y1 = targetPosition.y + targetNode.height / 2

  const xx1 = (x1 - x2) / (2 * w) - (y1 - y2) / (2 * h)
  const yy1 = (x1 - x2) / (2 * w) + (y1 - y2) / (2 * h)
  const a = 1 / (Math.abs(xx1) + Math.abs(yy1))
  const xx3 = a * xx1
  const yy3 = a * yy1
  const x = w * (xx3 + yy3) + x2
  const y = h * (-xx3 + yy3) + y2

  return { x, y }
}

// returns the position (top,right,bottom or right) passed node compared to the intersection point
function getEdgePosition(node: any, intersectionPoint: any) {
  const n = { ...node.positionAbsolute, ...node }
  const nx = Math.round(n.x)
  const ny = Math.round(n.y)
  const px = Math.round(intersectionPoint.x)
  const py = Math.round(intersectionPoint.y)

  if (px <= nx + 1) {
    return Position.Left
  }
  if (px >= nx + n.width - 1) {
    return Position.Right
  }
  if (py <= ny + 1) {
    return Position.Top
  }
  if (py >= n.y + n.height - 1) {
    return Position.Bottom
  }

  return Position.Top
}

// returns the parameters (sx, sy, tx, ty, sourcePos, targetPos) you need to create an edge
export function getEdgeParams(source: any, target: any) {
  const sourceIntersectionPoint = getNodeIntersection(source, target)
  const targetIntersectionPoint = getNodeIntersection(target, source)

  const sourcePos = getEdgePosition(source, sourceIntersectionPoint)
  const targetPos = getEdgePosition(target, targetIntersectionPoint)

  return {
    sx: sourceIntersectionPoint.x,
    sy: sourceIntersectionPoint.y,
    tx: targetIntersectionPoint.x,
    ty: targetIntersectionPoint.y,
    sourcePos,
    targetPos,
  }
}

const actionVerbs = [
  'Get',
  'List',
  'Put',
  'Delete',
  'Publish',
  'Detail',
  'Manage',
  'Read',
  'Write',
  'Enqueue',
  'Dequeue',
  'Access',
]

function verbFromNitricAction(action: string) {
  for (const verb of actionVerbs) {
    if (action.endsWith(verb)) {
      return verb
    }
  }

  return action
}

export function generateArchitectureData(data: WebSocketResponse): {
  nodes: Node[]
  edges: Edge[]
} {
  const nodes: Node[] = []
  const edges: Edge[] = []

  // Generate nodes from APIs
  data.apis.forEach((api) => {
    const apiAddress = data.apiAddresses[api.name]
    const routes = (api.spec && Object.keys(api.spec.paths)) || []

    const node = createNode<ApiNodeData>(api, 'api', {
      title: api.name,
      resource: api,
      icon: GlobeAltIcon,
      address: apiAddress,
      description: `${routes.length} ${
        routes.length === 1 ? 'Route' : 'Routes'
      }`,
    })

    const specEntries = (api.spec && api.spec.paths) || []

    Object.entries(specEntries).forEach(([path, operations]) => {
      AllHttpMethods.forEach((m) => {
        const method = operations && (operations[m] as any)

        if (!method) {
          return
        }

        edges.push({
          id: `e-${api.name}-${method.operationId}-${method['x-nitric-target']['name']}`,
          source: node.id,
          target: method['x-nitric-target']['name'],
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: 'auto-start-reverse',
          },
          label: 'routes',
        })
      })
    })

    nodes.push(node)
  })

  // Generate nodes from websockets
  data.websockets.forEach((ws) => {
    const wsAddress = data.websocketAddresses[ws.name]

    const events = Object.keys(ws.targets || {})

    const node = createNode<WebsocketNodeData>(ws, 'websocket', {
      title: ws.name,
      resource: ws,
      icon: ChatBubbleLeftRightIcon,
      description: `${events.length} ${
        events.length === 1 ? 'Event' : 'Events'
      }`,
      address: wsAddress,
    })

    const uniqueTargets = unique(
      events.map((trigger) => ({
        target: ws.targets[trigger as WebsocketEvent],
        trigger,
      })),
      (t) => t.target,
    )

    edges.push(
      ...uniqueTargets.map(({ target }) => {
        return {
          id: `e-${ws.name}-${target}`,
          source: node.id,
          target,
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: 'auto-start-reverse',
          },
          label: 'Triggers',
        }
      }),
    )

    nodes.push(node)
  })

  // Generate nodes from schedules
  data.schedules.forEach((schedule) => {
    const node = createNode<ScheduleNodeData>(schedule, 'schedule', {
      title: schedule.name,
      resource: schedule,
      icon: ClockIcon,
      description: ``,
      address: `${data.triggerAddress}/schedules/${schedule.name}`,
    })

    nodes.push(node)

    edges.push({
      id: `e-${schedule.name}-${schedule.target}`,
      source: node.id,
      target: schedule.target,
      animated: true,
      markerEnd: {
        type: MarkerType.ArrowClosed,
      },
      markerStart: {
        type: MarkerType.ArrowClosed,
        orient: 'auto-start-reverse',
      },
      label: 'Triggers',
    })
  })

  data.stores.forEach((store) => {
    const node = createNode<BucketNodeData>(store, 'keyvaluestore', {
      title: store.name,
      resource: store,
      icon: CircleStackIcon,
    })

    nodes.push(node)
  })

  data.secrets.forEach((secret) => {
    const node = createNode<BucketNodeData>(secret, 'secret', {
      title: secret.name,
      resource: secret,
      icon: LockClosedIcon,
    })

    nodes.push(node)
  })

  data.sqlDatabases.forEach((sql) => {
    const node = createNode<BucketNodeData>(sql, 'sql', {
      title: sql.name,
      resource: sql,
      icon: SiPostgresql,
    })

    edges.push(
      ...sql.requestingServices.map((target) => ({
        id: `e-${sql.name}-${target}`,
        source: node.id,
        target,
        animated: true,
        markerEnd: {
          type: MarkerType.ArrowClosed,
        },
        markerStart: {
          type: MarkerType.ArrowClosed,
          orient: 'auto-start-reverse',
        },
        label: 'Connects',
      })),
    )

    nodes.push(node)
  })

  data.queues.forEach((queue) => {
    const node = createNode<BucketNodeData>(queue, 'queue', {
      title: queue.name,
      resource: queue,
      icon: QueueListIcon,
    })

    nodes.push(node)
  })

  // Generate nodes from buckets
  data.buckets.forEach((bucket) => {
    const bucketNotifications = getBucketNotifications(
      bucket,
      data.notifications,
    )
    const node = createNode<BucketNodeData>(bucket, 'bucket', {
      title: bucket.name,
      resource: bucket,
      icon: ArchiveBoxIcon,
      description: `${bucketNotifications.length} ${
        bucketNotifications.length === 1 ? 'Notification' : 'Notifications'
      }`,
    })

    edges.push(
      ...bucketNotifications.map((notify) => {
        return {
          id: `e-${notify.bucket}-${notify.target}`,
          source: `bucket-${notify.bucket}`,
          target: notify.target,
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: 'auto-start-reverse',
          },
          label: 'Triggers',
        }
      }),
    )

    nodes.push(node)
  })

  // Generate nodes from buckets
  data.topics.forEach((topic) => {
    const subscriptions = getTopicSubscriptions(topic, data.subscriptions)

    const node = createNode<TopicNodeData>(topic, 'topic', {
      title: topic.name,
      resource: topic,
      icon: MegaphoneIcon,
      description: `${subscriptions.length} ${
        subscriptions.length === 1 ? 'Subscriber' : 'Subscribers'
      }`,
      address: `${data.triggerAddress}/topics/${topic.name}`,
    })
    nodes.push(node)

    const topicSubscriptions = data.subscriptions.filter(
      (sub) => sub.topic === topic.name,
    )

    edges.push(
      ...topicSubscriptions.map((subscription) => {
        return {
          id: `e-${subscription.topic}-${subscription.target}`,
          source: node.id,
          target: subscription.target,
          animated: true,
          markerEnd: {
            type: MarkerType.ArrowClosed,
          },
          markerStart: {
            type: MarkerType.ArrowClosed,
            orient: 'auto-start-reverse',
          },
          label: 'Triggers',
        } as Edge
      }),
    )
  })

  data.httpProxies.forEach((proxy) => {
    const proxyAddress = data.httpWorkerAddresses[proxy.name]

    const node = createNode<HttpProxyNodeData>(proxy, 'httpproxy', {
      title: `${proxyAddress.split(':')[2]}:${proxy.name.split(':')[1]}`,
      description: `Forwarding ${proxyAddress} to ${proxy.name}`,
      resource: proxy,
      icon: ArrowsRightLeftIcon,
      address: proxyAddress,
    })

    edges.push({
      id: `e-${proxy.name}-${proxy.target}`,
      source: `httpproxy-${proxy.name}`,
      target: proxy.target,
      animated: true,
      markerEnd: {
        type: MarkerType.ArrowClosed,
      },
      markerStart: {
        type: MarkerType.ArrowClosed,
        orient: 'auto-start-reverse',
      },
      label: 'Routes',
    })

    nodes.push(node)
  })

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
          orient: 'auto-start-reverse',
        },
        label: policy.actions.map(verbFromNitricAction).join(', '),
      } as Edge
    }),
  )

  data.services.forEach((service) => {
    const node: Node<ServiceNodeData> = {
      id: service.name,
      position: { x: 0, y: 0 },
      data: {
        title: service.name,
        description: '',
        resource: {
          filePath: service.filePath,
        },
        icon: CpuChipIcon,
        connectedEdges: [],
      },
      type: 'service',
    }

    const connectedEdges = getConnectedEdges([node], edges)
    node.data.connectedEdges = connectedEdges
    node.data.description =
      connectedEdges.length === 1
        ? `${connectedEdges.length} connection`
        : `${connectedEdges.length} connections`

    nodes.push(node)
  })

  if (import.meta.env.DEV) {
    console.log('nodes:', nodes)
    console.log('edges:', edges)
  }

  return { nodes, edges }
}
