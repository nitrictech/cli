import { FC, useMemo } from "react";
import type { WorkerResource } from "../../types";
import TreeView, { TreeItemType } from "../shared/TreeView";
import type { TreeItem, TreeItemIndex } from "react-complex-tree";

export type EventsTreeItemType = TreeItemType<WorkerResource>;

interface Props {
  resources: WorkerResource[];
  onSelect: (resource: WorkerResource) => void;
  initialItem: WorkerResource;
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
      if (!rootItems[resource.topicKey]) {
        rootItems[resource.topicKey] = {
          index: resource.topicKey,
          data: {
            label: resource.topicKey,
            data: resource,
          },
        };

        rootItem.children!.push(resource.topicKey);
      }
    }

    return rootItems;
  }, [resources]);

  return (
    <TreeView<EventsTreeItemType>
      label="Schedules"
      items={treeItems}
      initialItem={initialItem.topicKey}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data);
        }
      }}
      renderItemTitle={({ item }) => (
        <span className="truncate">{item.data.label}</span>
      )}
    />
  );
};

export default EventsTreeView;
