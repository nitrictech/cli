import { type ComponentType } from "react";

import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type ServiceNodeData = NodeBaseData<Record<string, any>>;

export const ServiceNode: ComponentType<NodeProps<ServiceNodeData>> = ({
  data,
  ...rest
}) => {
  const cleanedTitle = data.title.replace(/\\/g, "/");

  return (
    <NodeBase
      {...data}
      {...rest}
      title={cleanedTitle}
      drawerOptions={{
        title: `Details - ${cleanedTitle}`,
        description: data.description,
        children: <div className="flex flex-col">TODO</div>,
      }}
    />
  );
};
