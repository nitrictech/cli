import { type ComponentType } from "react";

import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type ServiceNodeData = NodeBaseData<Record<string, any>>;

export const ServiceNode: ComponentType<NodeProps<ServiceNodeData>> = ({
  data,
  selected,
}) => {
  const cleanedTitle = data.title.replace(/\\/g, "/");

  return (
    <NodeBase
      {...data}
      title={cleanedTitle}
      selected={selected}
      drawerOptions={{
        title: `Details - ${cleanedTitle}`,
        description: data.description,
        children: <div className="flex flex-col">TODO</div>,
      }}
    />
  );
};
