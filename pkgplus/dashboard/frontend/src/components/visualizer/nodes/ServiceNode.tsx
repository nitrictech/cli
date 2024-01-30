import { type ComponentType } from "react";

import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

type ServiceData = {
  filePath: string,
}

export type ServiceNodeData = NodeBaseData<ServiceData>;

export const ServiceNode: ComponentType<NodeProps<ServiceNodeData>> = (
  props
) => {
  const { data } = props;

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        children: <div className="flex flex-col">
          <a href={`vscode://file${data.resource.filePath}`}>Open in Vscode</a>
        </div>,
      }}
    />
  );
};
