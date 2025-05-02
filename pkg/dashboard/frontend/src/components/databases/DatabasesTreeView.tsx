import { type FC, useMemo } from 'react'
import type { SQLDatabase } from '../../types'
import TreeView, { type TreeItemType } from '../shared/TreeView'
import type { TreeItem, TreeItemIndex } from 'react-complex-tree'
import { Badge } from '../ui/badge'

export type DatabasesTreeItemType = TreeItemType<SQLDatabase>

interface Props {
  resources: SQLDatabase[]
  onSelect: (resource: SQLDatabase) => void
  initialItem: SQLDatabase
}

const DatabasesTreeView: FC<Props> = ({ resources, onSelect, initialItem }) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<DatabasesTreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: 'root',
      isFolder: true,
      children: [],
      data: null,
    }

    const rootItems: Record<TreeItemIndex, TreeItem<DatabasesTreeItemType>> = {
      root: rootItem,
    }

    for (const resource of resources) {
      // add api if not added already
      if (!rootItems[resource.name]) {
        rootItems[resource.name] = {
          index: resource.name,
          data: {
            label: resource.name,
            data: resource,
          },
        }

        rootItem.children!.push(resource.name)
      }
    }

    return rootItems
  }, [resources])

  return (
    <TreeView<DatabasesTreeItemType>
      label={'Databases'}
      items={treeItems}
      initialItem={initialItem.name}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data)
        }
      }}
      renderItemTitle={({ item }) => {
        return (
          <div className="flex w-full items-center justify-between">
            <span className="truncate text-foreground">{item.data.label}</span>
            {item.data.data?.status !== 'active' && (
              <span>
                <Badge className={'ml-2 bg-blue-600'}>
                  {item.data.data?.status}
                </Badge>
              </span>
            )}
          </div>
        )
      }}
    />
  )
}

export default DatabasesTreeView
