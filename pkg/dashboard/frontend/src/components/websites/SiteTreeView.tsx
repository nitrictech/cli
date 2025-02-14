import { type FC, useMemo } from 'react'
import TreeView, { type TreeItemType } from '../shared/TreeView'
import type { TreeItem, TreeItemIndex } from 'react-complex-tree'
import type { Website, Notification } from '@/types'
import { Tooltip, TooltipContent, TooltipTrigger } from '../ui/tooltip'
import { Badge } from '../ui/badge'
import { cn } from '@/lib/utils/cn'

export type SiteTreeItemType = TreeItemType<Website>

interface Props {
  websites: Website[]
  onSelect: (website: Website) => void
  initialItem: Website
}

const SiteTreeView: FC<Props> = ({ websites, onSelect, initialItem }) => {
  const treeItems: Record<
    TreeItemIndex,
    TreeItem<SiteTreeItemType>
  > = useMemo(() => {
    const rootItem: TreeItem = {
      index: 'root',
      isFolder: true,
      children: [],
      data: null,
    }

    const rootItems: Record<TreeItemIndex, TreeItem<SiteTreeItemType>> = {
      root: rootItem,
    }

    for (const website of websites) {
      // add api if not added already
      rootItems[website.name] = {
        index: website.name,
        data: {
          label: website.name,
          data: website,
        },
      }

      rootItem.children!.push(website.name)
    }

    return rootItems
  }, [websites])

  return (
    <TreeView<SiteTreeItemType>
      label="Websites"
      initialItem={initialItem.name}
      items={treeItems}
      getItemTitle={(item) => item.data.label}
      onPrimaryAction={(items) => {
        if (items.data.data) {
          onSelect(items.data.data)
        }
      }}
    />
  )
}

export default SiteTreeView
