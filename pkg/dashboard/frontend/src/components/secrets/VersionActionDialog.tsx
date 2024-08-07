import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { useSecret } from '@/lib/hooks/use-secret'
import { Loader2 } from 'lucide-react'
import { useRef, useState } from 'react'
import { useSecretsContext } from './SecretsContext'
import toast from 'react-hot-toast'

interface VersionActionDialogProps {
  action: 'add' | 'delete'
  open: boolean
  setOpen: (open: boolean) => void
}

export function VersionActionDialog({
  open,
  setOpen,
  action,
}: VersionActionDialogProps) {
  const { selectedSecret, selectedVersions } = useSecretsContext()
  const [loading, setLoading] = useState(false)
  const [value, setValue] = useState<string>('')
  const inputRef = useRef<HTMLInputElement>(null)
  const secretName = selectedSecret?.name

  const {
    addSecretVersion,
    deleteSecretVersion,
    mutate: refresh,
  } = useSecret(secretName)

  const handleSubmit = async () => {
    setLoading(true)

    if (action === 'add') {
      if (!value.trim()) {
        toast.error('Secret value is required')
        setLoading(false)
        inputRef.current?.focus()
        return
      }

      await addSecretVersion(value)
      setValue('')
    } else {
      if (!selectedVersions) {
        throw new Error('Selected versions are not provided')
      }

      await Promise.all([
        selectedVersions.map((version) => deleteSecretVersion(version)),
      ])
    }

    await new Promise((resolve) => setTimeout(resolve, 600))
    await refresh()

    setOpen(false)
    setLoading(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="leading-6">
            {action === 'add'
              ? `Add new version to ${secretName}`
              : `Are you sure that you want to delete the selected ${selectedVersions?.length} versions of ${secretName}?`}
          </DialogTitle>
          <DialogDescription>
            {action === 'add'
              ? `Input the new secret value.`
              : `Once deleted the versions cannot be recovered.`}
          </DialogDescription>
        </DialogHeader>
        <div className="py-4">
          <div className="grid grid-cols-4 items-center gap-4">
            {action === 'add' && (
              <>
                <Label
                  aria-required="true"
                  htmlFor="value"
                  className="text-right"
                >
                  Secret value
                </Label>
                <Input
                  ref={inputRef}
                  id="value"
                  data-testid="secret-value"
                  required
                  value={value}
                  onChange={(e) => setValue(e.target.value)}
                  placeholder="Secret value"
                  className="col-span-3"
                />
              </>
            )}
          </div>
        </div>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="ghost">Cancel</Button>
          </DialogClose>

          <Button
            onClick={handleSubmit}
            disabled={loading}
            data-testid="submit-secrets-dialog"
          >
            {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
            {action === 'add' ? 'Add New Version' : 'Delete selected versions'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
