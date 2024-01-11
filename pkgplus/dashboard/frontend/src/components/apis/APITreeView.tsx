import { type FC, useMemo } from "react";
import type { Endpoint, Method } from "../../types";
import { APIMethodBadge } from "./APIMethodBadge";
import TreeView, { type TreeItemType } from "../shared/TreeView";
import type { TreeItem, TreeItemIndex } from "react-complex-tree";

export interface APITreeItemType extends TreeItemType<Endpoint> {
  method?: Method;
}

interface Props {
  endpoints: Endpoint[];
  onSelect: (endpoint: Endpoint) => void;
  defaultTreeIndex: TreeItemIndex;
}

const APITreeView: FC<Props> = ({ endpoints, onSelect, defaultTreeIndex }) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<APITreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: "root",
      isFolder: true,
      children: [],
      data: null,
    };

    const rootItems: Record<TreeItemIndex, TreeItem<APITreeItemType>> = {
      root: rootItem,
    };

    for (const endpoint of endpoints) {
      // add api if not added already
      if (!rootItems[endpoint.api]) {
        rootItems[endpoint.api] = {
          index: endpoint.api,
          isFolder: true,
          children: [],
          data: {
            label: endpoint.api,
          },
        };

        rootItem.children!.push(endpoint.api);
      }

      // add each method of each path
      const id = endpoint.id;
      rootItems[id] = {
        index: id,
        data: {
          label: `${endpoint.method} - ${endpoint.path}`,
          method: endpoint.method,
          data: endpoint,
          parent: rootItems[endpoint.api],
        },
      };

      rootItems[endpoint.api].children?.push(id);
    }

    return rootItems;
  }, [endpoints]);

  return (
    <TreeView<APITreeItemType>
      label="API Explorer"
      initialItem={defaultTreeIndex}
      items={treeItems}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data);
        }
      }}
      renderItemTitle={({ title, item }) => (
        <>
          {!item.isFolder && item.data.method ? (
            <div className="grid w-full grid-cols-12 gap-4">
              <div className="col-span-5 flex">
                <APIMethodBadge method={item.data.method} />
              </div>
              <div className="col-span-7 flex justify-start">
                <span className="truncate">{item.data.data?.path}</span>
              </div>
            </div>
          ) : (
            <span>
              {item.children?.length
                ? `${title} (${item.children.length})`
                : title}
            </span>
          )}
        </>
      )}
    />
  );
};

export default APITreeView;
