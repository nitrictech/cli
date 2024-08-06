import { type FC, useMemo } from 'react'
import type { Secret } from '@/types'
import TreeView, { type TreeItemType } from '../shared/TreeView'
import type { TreeItem, TreeItemIndex } from 'react-complex-tree'

export type SecretsTreeItemType = TreeItemType<Secret>

interface Props {
  resources: Secret[]
  onSelect: (resource: Secret) => void
  initialItem: Secret
}

const SecretsTreeView: FC<Props> = ({ resources, onSelect, initialItem }) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<SecretsTreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: 'root',
      isFolder: true,
      children: [],
      data: null,
    }

    const rootItems: Record<TreeItemIndex, TreeItem<SecretsTreeItemType>> = {
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
    <TreeView<SecretsTreeItemType>
      label={'Secrets'}
      items={treeItems}
      initialItem={initialItem.name}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data)
        }
      }}
      renderItemTitle={({ item }) => {
        return <span className="truncate">{item.data.label}</span>
      }}
    />
  )
}

export default SecretsTreeView
