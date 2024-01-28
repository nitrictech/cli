import { type ComponentType } from "react";

import type { Schedule } from "@/types";
import type { NodeProps } from "reactflow";
import NodeBase, { type NodeBaseData } from "./NodeBase";

export type ScheduleNodeData = NodeBaseData<Schedule>;

export const ScheduleNode: ComponentType<NodeProps<ScheduleNodeData>> = ({
  data,
}) => {
  return (
    <NodeBase
      {...data}
      title={`${data.title} Schedule`}
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
