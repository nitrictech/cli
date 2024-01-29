import type { FC, PropsWithChildren } from "react";
import { Handle, Position, type NodeProps } from "reactflow";
import { DetailsDrawer } from "../DetailsDrawer";
import { cn } from "@/lib/utils";

export interface NodeBaseData<T = Record<string, any>> {
  resource: T;
  title: string;
  icon: React.ForwardRefExoticComponent<
    Omit<React.SVGProps<SVGSVGElement>, "ref"> & {
      title?: string | undefined;
      titleId?: string | undefined;
    } & React.RefAttributes<SVGSVGElement>
  >;
  iconClassName?: string;
  description?: string;
}

interface DrawerOptions extends PropsWithChildren {
  title: string;
  description?: string;
}

interface Props extends NodeProps<NodeBaseData> {
  drawerOptions?: DrawerOptions;
}

const NodeBase: FC<PropsWithChildren<Props>> = ({
  drawerOptions,
  selected,
  dragging,
  data: { icon: Icon, title, iconClassName },
}) => {
  return (
    <>
      {drawerOptions && (
        <DetailsDrawer {...drawerOptions} open={selected && !dragging} />
      )}
      <div className="overflow-hidden flex p-0.5 relative flex-grow rounded-md wrapper gradient">
        <div className="bg-white rounded items-center gap-4 pr-4 flex justify-center grow relative">
          <div className="h-full w-14 flex items-center justify-center">
            <div className="rounded-full gradient relative overflow-hidden size-10 flex items-center justify-center">
              <div className="z-10 bg-white size-9 rounded-full flex items-center justify-center">
                <Icon className={cn("size-6", iconClassName)} />
              </div>
            </div>
          </div>
          <div className="text-sm font-semibold">{title}</div>

          <Handle type="target" isConnectable={false} position={Position.Top} />
          <Handle
            type="source"
            isConnectable={false}
            position={Position.Bottom}
          />
        </div>
      </div>
    </>
  );
};

export default NodeBase;
