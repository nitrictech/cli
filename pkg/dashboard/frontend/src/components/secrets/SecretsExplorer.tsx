import { useWebSocket } from '../../lib/hooks/use-web-socket'
import type { Secret } from '@/types'
import { Loading } from '../shared'

import AppLayout from '../layout/AppLayout'
import BreadCrumbs from '../layout/BreadCrumbs'
import SecretsTreeView from './SecretsTreeView'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '../ui/select'
import { useEffect } from 'react'
import SecretVersionsTable from './SecretVersionsTable'
import { SecretsProvider, useSecretsContext } from './SecretsContext'
import NotFoundAlert from '../shared/NotFoundAlert'

const SecretsExplorer: React.FC = () => {
  const { data, loading } = useWebSocket()

  const { selectedSecret, setSelectedSecret } = useSecretsContext()

  useEffect(() => {
    if (!selectedSecret && data && data.secrets.length) {
      setSelectedSecret(data.secrets[0])
    }
  }, [data])

  const hasData = Boolean(data && data.secrets.length)

  return (
    <AppLayout
      title={'Secrets'}
      hideTitle
      routePath={`/secrets`}
      secondLevelNav={
        data &&
        selectedSecret && (
          <>
            <div className="flex min-h-12 items-center justify-between px-2 py-1">
              <span className="text-lg">Secrets</span>
            </div>
            <SecretsTreeView
              initialItem={selectedSecret}
              onSelect={setSelectedSecret}
              resources={data.secrets ?? []}
            />
          </>
        )
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {selectedSecret && hasData ? (
          <div className="flex max-w-[2000px] flex-col gap-8 md:pr-8">
            <div className="flex w-full flex-col gap-8">
              <div className="lg:hidden">
                <div className="ml-auto flex items-center"></div>
                {hasData && (
                  <Select
                    value={selectedSecret.name}
                    onValueChange={(name) => {
                      setSelectedSecret(
                        data?.secrets.find((b) => b.name === name),
                      )
                    }}
                  >
                    <SelectTrigger className="w-full">
                      <SelectValue placeholder={`Select Secret`} />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectGroup>
                        {data?.secrets.map((db) => (
                          <SelectItem key={db.name} value={db.name}>
                            {db.name}
                          </SelectItem>
                        ))}
                      </SelectGroup>
                    </SelectContent>
                  </Select>
                )}
              </div>
              <div className="space-y-4">
                <div className="hidden items-center gap-4 lg:flex">
                  <BreadCrumbs className="text-lg">
                    <span>Secrets</span>
                    <h2 className="font-body text-lg font-semibold">
                      {selectedSecret.name}
                    </h2>
                  </BreadCrumbs>
                </div>
                {!data?.secrets.some((s) => s.name === selectedSecret.name) && (
                  <NotFoundAlert>
                    Secret not found. It might have been updated or removed.
                    Select another secret.
                  </NotFoundAlert>
                )}
              </div>
              <SecretVersionsTable />
            </div>
          </div>
        ) : !hasData ? (
          <div>
            Please refer to our documentation on{' '}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/secrets#create-secrets"
              rel="noreferrer"
            >
              creating a secret
            </a>{' '}
            as we are unable to find any existing secrets.
          </div>
        ) : null}
      </Loading>
    </AppLayout>
  )
}

export default function SecretsExplorerWrapped() {
  return (
    <SecretsProvider>
      <SecretsExplorer />
    </SecretsProvider>
  )
}
