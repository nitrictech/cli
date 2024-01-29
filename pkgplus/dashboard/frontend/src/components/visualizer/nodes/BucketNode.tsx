import { type ComponentType } from "react";

import type { Bucket } from "@/types";
import type { Node, NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type BucketNodeData = NodeBaseData<Bucket>;

export const BucketNode: ComponentType<NodeProps<BucketNodeData>> = ({
  data,
}) => {
  return (
    <NodeBase
      {...data}
      title={`${data.title} Bucket`}
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
