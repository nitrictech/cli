import { type ComponentType } from "react";

import type { Topic } from "@/types";
import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type TopicNodeData = NodeBaseData<Topic>;

export const TopicNode: ComponentType<NodeProps<TopicNodeData>> = ({
  data,
}) => {
  return (
    <NodeBase
      {...data}
      title={`${data.title} Topic`}
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
