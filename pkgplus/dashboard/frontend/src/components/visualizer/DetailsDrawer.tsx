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
import { useState, type PropsWithChildren } from "react";
import ArrowLongRightIcon from "@heroicons/react/24/outline/ArrowLongRightIcon";
import { cn } from "@/lib/utils";
interface Props extends PropsWithChildren {
  title: string;
  description: string;
}

export const DetailsDrawer = ({ title, description, children }: Props) => {
  const [open, setOpen] = useState(false);

  return (
    <Drawer direction="right" open={open} onOpenChange={setOpen}>
      <button className="size-full" onClick={() => setOpen(true)}>
        <ArrowLongRightIcon
          className={cn("transition-all", open ? "-rotate-180" : "rotate-0")}
        />
      </button>

      <DrawerContent className="bg-white flex flex-col inset-auto rounded-l-[10px] rounded-r-none h-full w-[400px] mt-24 fixed bottom-0 right-0">
        <div className="mx-auto w-full max-w-sm p-4">
          <DrawerHeader>
            <DrawerTitle>{title}</DrawerTitle>
            <DrawerDescription>{description}</DrawerDescription>
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
