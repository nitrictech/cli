import React from 'react'
import type { Website } from '@/types'

interface SitesListProps {
  subsites: Website[]
  rootSite: Website
}

const SitesList: React.FC<SitesListProps> = ({ rootSite, subsites }) => {
  return (
    <div className="flex flex-col gap-y-1">
      <span className="font-bold">Root Site:</span>
      <div className="grid w-full grid-cols-12 gap-4">
        <div className="col-span-9 flex justify-start">
          <a
            target="_blank noreferrer noopener"
            className="truncate hover:underline"
            href={rootSite.url}
            rel="noreferrer"
          >
            {rootSite.url}
          </a>
        </div>
      </div>
      {subsites.length > 0 ? (
        <>
          <span className="font-bold">Subsites:</span>
          <div className="flex flex-col gap-y-2" data-testid="websites-list">
            {subsites.map((website) => (
              <div
                key={website.name}
                className="grid w-full grid-cols-12 gap-4"
              >
                <div className="col-span-9 flex justify-start">
                  <a
                    target="_blank noreferrer noopener"
                    className="truncate hover:underline"
                    href={website.url}
                    rel="noreferrer"
                  >
                    {website.url}
                  </a>
                </div>
              </div>
            ))}
          </div>
        </>
      ) : null}
    </div>
  )
}

export default SitesList
