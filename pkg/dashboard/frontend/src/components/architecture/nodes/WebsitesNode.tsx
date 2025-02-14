import { type ComponentType } from 'react'

import type { Website } from '@/types'
import type { NodeProps } from 'reactflow'
import NodeBase, { type NodeBaseData } from './NodeBase'
import SitesList from '@/components/websites/SitesList'

export type WebsitesNodeData = NodeBaseData<Website[]>

export const WebsitesNode: ComponentType<NodeProps<WebsitesNodeData>> = (
  props,
) => {
  const { data } = props

  const websites = data.resource

  const rootWebsite = websites.find((website) =>
    /localhost:\d+$/.test(website.url.replace(/\/$/, '')),
  )

  const description = `${websites.length} websites stored in a bucket and served via CDN.`

  return (
    <NodeBase
      {...props}
      drawerOptions={{
        title: `Websites`,
        description,
        icon: data.icon,
        nodeType: 'websites',
        testHref: '/websites',
        trailingChildren:
          websites && rootWebsite ? (
            <SitesList
              rootSite={rootWebsite}
              subsites={websites.filter((site) => site !== rootWebsite)}
            />
          ) : null,
      }}
    />
  )
}
