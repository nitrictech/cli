import { type ComponentType } from "react";

import type { Api } from "@/types";
import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type ApiNodeData = NodeBaseData<Api>;

export const APINode: ComponentType<NodeProps<ApiNodeData>> = ({
  data,
  ...rest
}) => {
  return (
    <NodeBase
      {...data}
      {...rest}
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
