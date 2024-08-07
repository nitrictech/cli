import { useCallback } from 'react'
import useSWR from 'swr'
import { fetcher } from './fetcher'
import type { SecretVersion } from '@/types'
import { SECRETS_API } from '../constants'

export const useSecret = (secretName?: string) => {
  const { data, mutate } = useSWR<SecretVersion[]>(
    secretName
      ? `${SECRETS_API}?action=list-versions&secret=${secretName}`
      : null,
    fetcher(),
  )

  const addSecretVersion = useCallback(
    async (value: string) => {
      return fetch(
        `${SECRETS_API}?action=add-secret-version&secret=${secretName}`,
        {
          method: 'POST',
          body: JSON.stringify({ value }),
        },
      )
    },
    [secretName],
  )

  const deleteSecretVersion = useCallback(
    async (sv: SecretVersion) => {
      return fetch(
        `${SECRETS_API}?action=delete-secret&secret=${secretName}&version=${sv.version}&latest=${sv.latest}`,
        {
          method: 'DELETE',
        },
      )
    },
    [secretName],
  )

  return {
    data,
    mutate,
    addSecretVersion,
    deleteSecretVersion,
    loading: !data,
  }
}
