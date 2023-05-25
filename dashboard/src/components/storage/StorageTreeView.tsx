import { FC, useMemo } from "react";
import TreeView, { TreeItemType } from "../shared/TreeView";
import type { TreeItem, TreeItemIndex } from "react-complex-tree";

export type StorageTreeItemType = TreeItemType<string>;

interface Props {
  buckets: string[];
  onSelect: (bucket: string) => void;
  initialItem: string;
}

const StorageTreeView: FC<Props> = ({ buckets, onSelect, initialItem }) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<StorageTreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: "root",
      isFolder: true,
      children: [],
      data: null,
    };

    const rootItems: Record<TreeItemIndex, TreeItem<StorageTreeItemType>> = {
      root: rootItem,
    };

    for (const bucket of buckets) {
      // add api if not added already
      rootItems[bucket] = {
        index: bucket,
        data: {
          label: bucket,
        },
      };

      rootItem.children!.push(bucket);
    }

    return rootItems;
  }, [buckets]);

  return (
    <TreeView<StorageTreeItemType>
      label="Buckets"
      initialItem={initialItem}
      items={treeItems}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.label) {
          onSelect(items.data.label);
        }
      }}
      renderItemTitle={({ item }) => (
        <span className="truncate">{item.data.label}</span>
      )}
    />
  );
};

export default StorageTreeView;
