import { type FC, useMemo } from "react";
import type { Schedule, Topic } from "../../types";
import TreeView, { type TreeItemType } from "../shared/TreeView";
import type { TreeItem, TreeItemIndex } from "react-complex-tree";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";
import { Badge } from "../ui/badge";
import { cn } from "@/lib/utils";

export type EventsTreeItemType = TreeItemType<Schedule | Topic>;

interface Props {
  resources: (Schedule | Topic)[];
  onSelect: (resource: Schedule | Topic) => void;
  initialItem: Schedule | Topic;
}

const EventsTreeView: FC<Props> = ({ resources, onSelect, initialItem }) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<EventsTreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: "root",
      isFolder: true,
      children: [],
      data: null,
    };

    const rootItems: Record<TreeItemIndex, TreeItem<EventsTreeItemType>> = {
      root: rootItem,
    };

    for (const resource of resources) {
      // add api if not added already
      if (!rootItems[resource.name]) {
        rootItems[resource.name] = {
          index: resource.name,
          data: {
            label: resource.name,
            data: resource,
          },
        };

        rootItem.children!.push(resource.name);
      }
    }

    return rootItems;
  }, [resources]);

  return (
    <TreeView<EventsTreeItemType>
      label="Schedules"
      items={treeItems}
      initialItem={initialItem.name}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data);
        }
      }}
      renderItemTitle={({ item }) => {
        const topicSubscriberCount =
          typeof (item.data.data as Topic)?.subscriberCount == "number"
            ? (item.data.data as Topic)?.subscriberCount
            : null;

        return (
          <span className="truncate">
            {topicSubscriberCount ? (
              <>
                {item.data.label}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Badge
                        className={cn(
                          "ml-2",
                          topicSubscriberCount > 0
                            ? "bg-blue-600"
                            : "bg-orange-400"
                        )}
                      >
                        {topicSubscriberCount}
                      </Badge>
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <p>{topicSubscriberCount} subscribers to this topic</p>
                  </TooltipContent>
                </Tooltip>
              </>
            ) : (
              item.data.label
            )}
          </span>
        );
      }}
    />
  );
};

export default EventsTreeView;
