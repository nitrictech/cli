import { useId, useState } from 'react'
import {
  ControlledTreeEnvironment,
  type ControlledTreeEnvironmentProps,
  Tree,
  type TreeItem,
  type TreeItemIndex,
} from 'react-complex-tree'
import {
  FolderIcon,
  FolderOpenIcon,
  MagnifyingGlassIcon,
} from '@heroicons/react/24/outline'
import { cn } from '@/lib/utils'
import { debounce } from 'radash'
import TextField from './TextField'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'

interface Props<T>
  extends Omit<ControlledTreeEnvironmentProps<T>, 'viewState'> {
  label: string
  initialItem?: TreeItemIndex
}

export interface TreeItemType<T> {
  label: string
  data?: T
  parent?: TreeItem<TreeItemType<T>>
}

const filterData = <T extends Record<string, any>>(
  data: Record<TreeItemIndex, TreeItem<T>>,
  query: string,
) => {
  if (!query) {
    return data
  }

  const filteredKeys = Object.keys(data).filter(
    (key) =>
      key === 'root' ||
      data[key].isFolder ||
      key.toLowerCase().includes(query.toLowerCase()),
  )

  const filteredObject: Record<TreeItemIndex, TreeItem<T>> = {}
  filteredKeys.forEach((key) => {
    filteredObject[key] = {
      ...data[key],
    }
  })

  // filter out children
  Object.entries(filteredObject).forEach(([key, value]) => {
    if (value.children) {
      value.children = value.children.filter((c) =>
        filteredKeys.includes(c.toString()),
      )

      if (value.children.length === 0) {
        delete filteredObject[key]
      }
    }
  })

  return filteredObject
}

const TreeView = <T extends Record<string, any>>({
  label,
  items,
  initialItem,
  ...props
}: Props<T>) => {
  const id = useId()
  const [searchQuery, setSearchQuery] = useState('')
  const [focusedItem, setFocusedItem] = useState<TreeItemIndex>()
  const [expandedItems, setExpandedItems] = useState<TreeItemIndex[]>(
    initialItem && items[initialItem] && items[initialItem].data?.parent?.index
      ? [items[initialItem].data.parent?.index]
      : [],
  )
  const [selectedItems, setSelectedItems] = useState<TreeItemIndex[]>(
    initialItem ? [initialItem] : [],
  )

  const debouncedSearch = debounce({ delay: 100 }, (search: string) => {
    setSearchQuery(search)
  })

  const filteredItems = filterData(items, searchQuery)

  return (
    <div className="flex flex-col gap-2">
      <div className="px-2">
        <TextField
          id="tree-search"
          label="Search"
          hideLabel
          icon={MagnifyingGlassIcon}
          placeholder="Search"
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
              <FolderOpenIcon className="h-6 w-6 text-primary" />
            ) : (
              <FolderIcon className="h-6 w-6 text-gray-500" />
            )
          ) : null
        }
        renderItem={({ title, arrow, depth, context, children, item }) => {
          const buttonContent = (
            <button
              {...context.itemContainerWithoutChildrenProps}
              {...context.interactiveElementProps}
              type="button"
              className={cn(
                'flex w-full items-center justify-start gap-2 p-2 px-4 transition-colors hover:bg-gray-200',
                context.isSelected && 'bg-gray-200',
                depth === 0 && context.isExpanded ? 'pb-2' : '',
              )}
              style={{
                paddingLeft: depth > 0 ? depth * 15 : undefined,
              }}
            >
              {arrow}
              {title}
            </button>
          )

          return (
            <li
              {...context.itemContainerWithChildrenProps}
              className={cn('flex w-full flex-col items-start text-base')}
            >
              {/* if text is over 30 chars, show tooltip */}
              {item.data.label.length >= 30 ? (
                <Tooltip disableHoverableContent>
                  <TooltipTrigger asChild>{buttonContent}</TooltipTrigger>
                  <TooltipContent side="bottom">
                    <p>{item.data.label}</p>
                  </TooltipContent>
                </Tooltip>
              ) : (
                buttonContent
              )}
              {children}
            </li>
          )
        }}
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
              (expandedItemIndex) => expandedItemIndex !== item.index,
            ),
          )
        }
        onSelectItems={(items) => setSelectedItems(items)}
        getItemTitle={(item) => item.data.label}
      >
        <Tree treeId={id} rootItem="root" treeLabel={label} />
      </ControlledTreeEnvironment>
    </div>
  )
}

export default TreeView
