import { useSecret } from '@/lib/hooks/use-secret'
import type { Secret, SecretVersion } from '@/types'
import React, { createContext, useState, type PropsWithChildren } from 'react'
import { VersionActionDialog } from './VersionActionDialog'

interface SecretsContextProps {
  selectedVersions: SecretVersion[]
  setSelectedVersions: React.Dispatch<React.SetStateAction<SecretVersion[]>>
  selectedSecret?: Secret
  setSelectedSecret: (secret: Secret | undefined) => void
  setDialogAction: React.Dispatch<React.SetStateAction<'add' | 'delete'>>
  setDialogOpen: React.Dispatch<React.SetStateAction<boolean>>
}

export const SecretsContext = createContext<SecretsContextProps>({
  selectedVersions: [],
  setSelectedVersions: () => {},
  selectedSecret: undefined,
  setSelectedSecret: () => {},
  setDialogAction: () => {},
  setDialogOpen: () => {},
})

export const SecretsProvider: React.FC<PropsWithChildren> = ({ children }) => {
  const [selectedSecret, setSelectedSecret] = useState<Secret>()

  const [selectedVersions, setSelectedVersions] = useState<SecretVersion[]>([])
  const [dialogOpen, setDialogOpen] = useState(false)
  const [dialogAction, setDialogAction] = useState<'add' | 'delete'>('add')

  return (
    <SecretsContext.Provider
      value={{
        selectedVersions,
        setSelectedVersions,
        selectedSecret,
        setSelectedSecret,
        setDialogAction,
        setDialogOpen,
      }}
    >
      {selectedSecret && (
        <VersionActionDialog
          action={dialogAction}
          open={dialogOpen}
          setOpen={setDialogOpen}
        />
      )}
      {children}
    </SecretsContext.Provider>
  )
}

export const useSecretsContext = () => {
  return React.useContext(SecretsContext)
}
