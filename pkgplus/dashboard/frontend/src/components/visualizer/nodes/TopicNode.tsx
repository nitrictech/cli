import { type ComponentType } from "react";

import type { Topic } from "@/types";
import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type TopicNodeData = NodeBaseData<Topic>;

export const TopicNode: ComponentType<NodeProps<TopicNodeData>> = (props) => {
  const { data } = props;

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Details - ${data.title}`,
        description: data.description,
        testHref: `/topics`, // TODO add url param to switch to resource
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
