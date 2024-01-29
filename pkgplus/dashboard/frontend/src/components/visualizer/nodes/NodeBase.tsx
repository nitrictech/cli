import type { FC, PropsWithChildren, ReactNode } from "react";
import { Handle, Position } from "reactflow";
import { DetailsDrawer } from "../DetailsDrawer";

export interface NodeBaseData<T = Record<string, any>> {
  resource: T;
  title: string;
  icon: React.ForwardRefExoticComponent<
    Omit<React.SVGProps<SVGSVGElement>, "ref"> & {
      title?: string | undefined;
      titleId?: string | undefined;
    } & React.RefAttributes<SVGSVGElement>
  >;
  description: string;
}

interface DrawerOptions extends PropsWithChildren {
  title: string;
  description: string;
}

interface Props extends NodeBaseData {
  drawerOptions?: DrawerOptions;
}

const NodeBase: FC<PropsWithChildren<Props>> = ({
  drawerOptions,
  icon: Icon,
  title,
  description,
}) => {
  return (
    <>
      <div className="gradient rounded-full w-6 h-6 top-0 p-0.5 shadow z-10 overflow-hidden absolute right-0 translate-x-[45%] -translate-y-1/2">
        <div className="bg-white flex items-center justify-center relative rounded-full grow">
          <Icon className="size-full" />
        </div>
      </div>

      {drawerOptions && (
        <div className="gradient nitric-remove-on-share rounded-full w-6 h-6 bottom-0 p-0.5 shadow z-10 overflow-hidden absolute right-0 translate-x-[45%] translate-y-1/2">
          <div className="bg-white flex items-center justify-center relative rounded-full grow">
            <DetailsDrawer {...drawerOptions} />
          </div>
        </div>
      )}
      <div className="overflow-hidden flex p-0.5 relative flex-grow rounded-md wrapper gradient">
        <div className="bg-white rounded p-4 flex flex-col justify-center grow relative">
          <div className="flex flex-col gap-y-1">
            <div className="text-sm font-semibold">{title}</div>
            {description && (
              <div className="text-xs text-gray-500">{description}</div>
            )}
          </div>
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
