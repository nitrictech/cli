import { type ComponentType } from "react";

import type { APIDoc, Api } from "@/types";
import type { Node, NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";
import GlobeAltIcon from "@heroicons/react/24/outline/GlobeAltIcon";

type ApiNodeData = NodeBaseData<Api>;

type ApiNode = Node<ApiNodeData>;

export const createApiNode = (api: Api) => {
  const routes = Object.keys(api.spec.paths);
  return {
    id: api.name,
    position: { x: 0, y: 0 },
    data: {
      title: api.name,
      resource: api,
      icon: GlobeAltIcon,
      description: `An API with ${routes.length} ${
        routes.length === 1 ? "Route" : "Routes"
      }`,
    },
    type: "api",
  } as ApiNode;
};

export const APINode: ComponentType<NodeProps<ApiNode["data"]>> = ({
  data,
}) => {
  return (
    <NodeBase
      {...data}
      title={`${data.title} API`}
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
