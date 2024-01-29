import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
} from "../ui/drawer";
import { Button } from "../ui/button";
import type { PropsWithChildren } from "react";
import { useStoreApi } from "reactflow";
interface Props extends PropsWithChildren {
  title: string;
  description?: string;
  open: boolean;
}

export const DetailsDrawer = ({
  title,
  description,
  children,
  open,
}: Props) => {
  const store = useStoreApi();

  return (
    <Drawer
      direction="right"
      open={open}
      onOpenChange={(open) => {
        if (!open) {
          const { unselectNodesAndEdges } = store.getState();
          unselectNodesAndEdges();
        }
      }}
    >
      <DrawerContent className="bg-white flex flex-col inset-auto rounded-l-[10px] rounded-r-none h-full w-[400px] mt-24 fixed bottom-0 right-0">
        <div className="mx-auto w-full max-w-sm p-4">
          <DrawerHeader>
            <DrawerTitle>{title}</DrawerTitle>
            {description && (
              <DrawerDescription>{description}</DrawerDescription>
            )}
          </DrawerHeader>
          <div className="p-4">{children}</div>
          <DrawerFooter>
            <DrawerClose asChild>
              <Button variant="outline">Close</Button>
            </DrawerClose>
          </DrawerFooter>
        </div>
      </DrawerContent>
    </Drawer>
  );
};
