import { FC, useMemo } from "react";
import type { WebSocket } from "../../types";
import TreeView, { TreeItemType } from "../shared/TreeView";
import type { TreeItem, TreeItemIndex } from "react-complex-tree";

export type EventsTreeItemType = TreeItemType<WebSocket>;

interface Props {
  websockets: WebSocket[];
  onSelect: (resource: WebSocket) => void;
  initialItem: WebSocket;
}

const WSTreeView: FC<Props> = ({ websockets, onSelect, initialItem }) => {
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

    for (const resource of websockets) {
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
  }, [websockets]);

  return (
    <TreeView<EventsTreeItemType>
      label="Websockets"
      items={treeItems}
      initialItem={initialItem.name}
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

export default WSTreeView;
