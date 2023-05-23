import { FC, useId, useRef, useState } from "react";
//import "react-complex-tree/lib/style-modern.css";
import {
  ControlledTreeEnvironment,
  ControlledTreeEnvironmentProps,
  Tree,
  TreeItem,
  TreeItemIndex,
} from "react-complex-tree";
import {
  FolderIcon,
  FolderOpenIcon,
  MagnifyingGlassIcon,
} from "@heroicons/react/24/outline";
import classNames from "classnames";
import { debounce } from "radash";
import TextField from "./TextField";

interface Props<T>
  extends Omit<ControlledTreeEnvironmentProps<T>, "viewState"> {
  label: string;
  initialItem?: TreeItemIndex;
}

export interface TreeItemType<T> {
  label: string;
  data?: T;
  parent?: TreeItem<TreeItemType<T>>;
}

const filterData = <T extends Record<string, any>>(
  data: Record<TreeItemIndex, TreeItem<T>>,
  query: string
) => {
  if (!query) {
    return data;
  }

  const filteredKeys = Object.keys(data).filter(
    (key) =>
      key === "root" ||
      data[key].isFolder ||
      key.toLowerCase().includes(query.toLowerCase())
  );

  const filteredObject: Record<TreeItemIndex, TreeItem<T>> = {};
  filteredKeys.forEach((key) => {
    filteredObject[key] = {
      ...data[key],
    };
  });

  // filter out children
  Object.entries(filteredObject).forEach(([key, value]) => {
    if (value.children) {
      value.children = value.children.filter((c) =>
        filteredKeys.includes(c.toString())
      );

      if (value.children.length === 0) {
        delete filteredObject[key];
      }
    }
  });

  return filteredObject;
};

const TreeView = <T extends Record<string, any>>({
  label,
  items,
  initialItem,
  ...props
}: Props<T>) => {
  const id = useId();
  const [searchQuery, setSearchQuery] = useState("");
  const [focusedItem, setFocusedItem] = useState<TreeItemIndex>();
  const [expandedItems, setExpandedItems] = useState<TreeItemIndex[]>(
    initialItem && items[initialItem].data.parent?.index
      ? [items[initialItem].data.parent?.index]
      : []
  );
  const [selectedItems, setSelectedItems] = useState<TreeItemIndex[]>(
    initialItem ? [initialItem] : []
  );

  const debouncedSearch = debounce({ delay: 100 }, (search: string) => {
    setSearchQuery(search);
  });

  const filteredItems = filterData(items, searchQuery);

  return (
    <div className="flex flex-col gap-2">
      <div className="px-2">
        <TextField
          id="tree-search"
          label="Search"
          hideLabel
          icon={MagnifyingGlassIcon}
          onChange={(event) => debouncedSearch(event.target.value)}
        />
      </div>
      <ControlledTreeEnvironment<T>
        renderItemsContainer={({ children, containerProps }) => (
          <ul {...containerProps} role="group" className="w-full">
            {children}
          </ul>
        )}
        showLiveDescription={false}
        renderItemArrow={({ item, context }) =>
          item.isFolder ? (
            context.isExpanded ? (
              <FolderOpenIcon className="w-6 h-6 text-blue-600" />
            ) : (
              <FolderIcon className="w-6 h-6 text-gray-500" />
            )
          ) : null
        }
        renderItem={({ title, arrow, depth, context, children, item }) => (
          <li
            {...context.itemContainerWithChildrenProps}
            className={classNames("flex w-full flex-col text-base items-start")}
          >
            <button
              {...context.itemContainerWithoutChildrenProps}
              {...context.interactiveElementProps}
              type="button"
              className={classNames(
                "flex w-full items-center justify-start gap-2 p-2 px-4 hover:bg-gray-200 transition-colors",
                context.isSelected && "bg-gray-200",
                depth === 0 && context.isExpanded ? "pb-2" : ""
              )}
              style={{
                paddingLeft: depth > 0 ? depth * 15 : undefined,
              }}
              title={item.data.label}
            >
              {arrow}
              {title}
            </button>
            {children}
          </li>
        )}
        items={filteredItems}
        {...props}
        viewState={{
          [id]: {
            focusedItem,
            expandedItems,
            selectedItems,
          },
        }}
        onFocusItem={(item) => setFocusedItem(item.index)}
        onExpandItem={(item) =>
          setExpandedItems([...expandedItems, item.index])
        }
        onCollapseItem={(item) =>
          setExpandedItems(
            expandedItems.filter(
              (expandedItemIndex) => expandedItemIndex !== item.index
            )
          )
        }
        onSelectItems={(items) => setSelectedItems(items)}
        getItemTitle={(item) => item.data.label}
      >
        <Tree treeId={id} rootItem="root" treeLabel={label} />
      </ControlledTreeEnvironment>
    </div>
  );
};

export default TreeView;
