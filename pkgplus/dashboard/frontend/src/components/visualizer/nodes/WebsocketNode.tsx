import { type ComponentType } from "react";

import type { WebSocket } from "@/types";
import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type WebsocketNodeData = NodeBaseData<WebSocket>;

export const WebsocketNode: ComponentType<NodeProps<WebsocketNodeData>> = ({
  data,
}) => {
  return (
    <NodeBase
      {...data}
      title={`${data.title} Websocket`}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        children: (
          <div className="flex flex-col">
            <span className="font-bold">Requested by:</span>
            <span>{data.resource.requestingServices.join(", ")}</span>
          </div>
        ),
      }}
    />
  );
};
