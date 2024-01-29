import { type ComponentType } from "react";

import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type ServiceNodeData = NodeBaseData<Record<string, any>>;

export const ServiceNode: ComponentType<NodeProps<ServiceNodeData>> = ({
  data,
  selected,
}) => {
  return (
    <NodeBase
      {...data}
      selected={selected}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        children: <div className="flex flex-col">TODO</div>,
      }}
    />
  );
};
