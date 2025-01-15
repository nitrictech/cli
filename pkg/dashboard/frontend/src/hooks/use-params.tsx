import {
  createContext,
  useContext,
  useMemo,
  useCallback,
  useState,
  useEffect,
  type ReactNode,
} from 'react'

type ParamsContextValue = {
  setParams: (name: string, value: string | null) => void
  searchParams: URLSearchParams
}

const ParamsContext = createContext<ParamsContextValue | null>(null)

type ParamsProviderProps = {
  children: ReactNode
}

export const ParamsProvider = ({ children }: ParamsProviderProps) => {
  const [search, setSearch] = useState(window.location.search)
  const pathname = window.location.pathname

  useEffect(() => {
    const handlePopState = () => {
      setSearch(window.location.search)
    }

    window.addEventListener('popstate', handlePopState)
    return () => {
      window.removeEventListener('popstate', handlePopState)
    }
  }, [])

  const setParams = useCallback(
    (name: string, value: string | null) => {
      const latestSearchParams = new URLSearchParams(window.location.search)

      if (!value) {
        latestSearchParams.delete(name)
      } else {
        latestSearchParams.set(name, value)
      }

      const updatedSearch = latestSearchParams.toString()
      const url = updatedSearch ? `${pathname}?${updatedSearch}` : pathname

      window.history.pushState(null, '', url)
      setSearch(updatedSearch ? `?${updatedSearch}` : '')
    },
    [search, pathname],
  )

  const value = useMemo(
    () => ({
      setParams,
      searchParams: new URLSearchParams(search),
    }),
    [setParams, search],
  )

  return (
    <ParamsContext.Provider value={value}>{children}</ParamsContext.Provider>
  )
}

export const useParams = () => {
  const context = useContext(ParamsContext)
  if (!context) {
    throw new Error('useParams must be used within a ParamsProvider')
  }
  return context
}
