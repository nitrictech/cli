import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import Badge from './Badge'
import { getDateString } from '@/lib/utils/get-date-string'
import { createElement } from 'react'

interface HistoryAccordionItem {
  status?: number
  success?: boolean
  time: number
  label: string
  content?: React.ReactNode
}

interface Props {
  items: HistoryAccordionItem[]
}

const DivElement = (props: React.HTMLProps<HTMLDivElement>) => (
  <div {...props} />
)

const HistoryAccordion = ({ items }: Props) => {
  return (
    <ScrollArea className="h-[30rem]" type="always">
      <Accordion type="multiple" className="mx-2 my-2 flex flex-col">
        {items.map((item, idx) => {
          const TriggerElement = item.content ? AccordionTrigger : DivElement

          return (
            <AccordionItem key={idx} value={idx.toString()}>
              <TriggerElement className="p-2 !no-underline hover:bg-primary/5">
                <div className="flex w-full flex-row items-center gap-4 font-body">
                  {typeof item.success === 'boolean' ? (
                    <Badge
                      status={item.success ? 'green' : 'red'}
                      className="!text-md h-6 w-12 sm:w-20"
                    >
                      {item.success ? 'success' : 'failure'}
                    </Badge>
                  ) : (
                    item.status && (
                      <Badge
                        status={item.status < 400 ? 'green' : 'red'}
                        className="!text-md h-6 w-12 sm:w-20"
                      >
                        <span className="mr-0.5 hidden md:inline">
                          Status:{' '}
                        </span>
                        {item.status}
                      </Badge>
                    )
                  )}
                  <p className="max-w-[200px] truncate text-sm md:max-w-lg">
                    {item.label}
                  </p>
                  <p className="ml-auto hidden pr-2 text-sm sm:inline">
                    {getDateString(item.time)}
                  </p>
                </div>
              </TriggerElement>
              {item.content && (
                <AccordionContent className="pb-2 text-sm text-gray-500">
                  <div className="flex flex-col py-4">
                    <div className="bg-white sm:rounded-lg">{item.content}</div>
                  </div>
                </AccordionContent>
              )}
            </AccordionItem>
          )
        })}
      </Accordion>
    </ScrollArea>
  )
}

export default HistoryAccordion
