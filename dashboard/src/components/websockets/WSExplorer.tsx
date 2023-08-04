import { useEffect, useRef, useState } from "react";
import type { APIRequest, WebSocket, WebSocketsInfo } from "../../types";
import { FieldRows, Loading } from "../shared";
import { formatJSON, generatePath, getHost } from "../../lib/utils";

import { useWebSocket } from "../../lib/hooks/use-web-socket";
import AppLayout from "../layout/AppLayout";
import WSTreeView from "./WSTreeView";
import { copyToClipboard } from "../../lib/utils/copy-to-clipboard";
import toast from "react-hot-toast";
import {
  CheckCircleIcon,
  ClipboardIcon,
  ArrowDownCircleIcon,
  ArrowUpCircleIcon,
  ExclamationCircleIcon,
  InformationCircleIcon,
  TrashIcon,
} from "@heroicons/react/24/outline";
import { Button } from "../ui/button";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "../ui/accordion";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../ui/select";
import format from "date-fns/format";

import { Input } from "../ui/input";
import { ScrollArea } from "../ui/scroll-area";
import CodeEditor from "../apis/CodeEditor";
import {
  Card,
  CardContent,
  CardFooter,
  CardHeader,
  CardTitle,
} from "../ui/card";
import { Textarea } from "../ui/textarea";
import useSWRSubscription from "swr/subscription";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs";
import { Badge } from "../ui/badge";
import { Tooltip, TooltipContent, TooltipTrigger } from "../ui/tooltip";

export const LOCAL_STORAGE_KEY = "nitric-local-dash-api-history";

interface Message {
  ts: number;
  data: any;
  type: "connect" | "disconnect" | "message-in" | "message-out" | "error";
}

const MessageIcon = ({ type }: Pick<Message, "type">) => {
  const className = "w-6 h-6 mr-1";

  switch (type) {
    case "connect":
      return <CheckCircleIcon className={`${className} text-green-500`} />;
    case "error":
      return <ExclamationCircleIcon className={`${className} text-red-500`} />;
    case "message-in":
      return <ArrowDownCircleIcon className={`${className} text-blue-500`} />;
    case "message-out":
      return <ArrowUpCircleIcon className={`${className} text-orange-500`} />;
  }

  return <InformationCircleIcon className={className} />;
};

const WSExplorer = () => {
  const { data, loading } = useWebSocket();
  const websocketRef = useRef<globalThis.WebSocket>();
  const [messages, setMessages] = useState<Message[]>([]);
  const [currentPayload, setCurrentPayload] = useState<string>();
  const [payloadType, setPayloadType] = useState("text");
  const [monitorMessageFilter, setMonitorMessageFilter] = useState("");
  const [messageFilter, setMessageFilter] = useState("");
  const [messageTypeFilter, setMessageTypeFilter] = useState("all");
  const [tab, setTab] = useState("monitor");
  const [selectedWebsocket, setselectedWebsocket] = useState<WebSocket>();

  const [connected, setConnected] = useState(false);
  const [queryParams, setQueryParams] = useState<APIRequest["queryParams"]>([
    {
      key: "",
      value: "",
    },
  ]);

  useEffect(() => {
    if (!selectedWebsocket && data?.websockets.length) {
      setselectedWebsocket(data?.websockets[0]);
    }
  }, [data?.websockets]);

  const websocketAddress =
    selectedWebsocket && data?.websocketAddresses[selectedWebsocket?.name]
      ? `ws://${generatePath(
          data?.websocketAddresses[selectedWebsocket?.name],
          [],
          queryParams
        )}`
      : "";

  const host = getHost();

  const { data: wsData, error } = useSWRSubscription<WebSocketsInfo>(
    host ? `ws://${host}/ws-info` : null,
    (key: any, { next }: any) => {
      const socket = new WebSocket(key);

      socket.addEventListener("message", (event) => {
        const message = JSON.parse(event.data) as WebSocketsInfo;

        next(null, message);
      });

      socket.addEventListener("error", (event: any) => next(event.error));
      return () => socket.close();
    }
  );

  const wsInfo =
    selectedWebsocket && wsData ? wsData![selectedWebsocket?.name] : undefined;

  useEffect(() => {
    if (websocketAddress && connected) {
      const socket = new WebSocket(websocketAddress);

      // set socket ref
      websocketRef.current = socket;

      socket.addEventListener("message", (event) => {
        console.log(event);
        setMessages((prev) => [
          {
            data: event.data,
            ts: new Date().getTime(),
            type: "message-in",
          },
          ...prev,
        ]);
      });
      socket.addEventListener("error", (event: any) => {
        setMessages((prev) => [
          {
            data: event.error,
            ts: new Date().getTime(),
            type: "error",
          },
          ...prev,
        ]);
      });
      // Event listener to handle connection open
      socket.addEventListener("open", (event) => {
        console.log(event);
        setMessages((prev) => [
          {
            data: `Connected to ${websocketAddress}`,
            ts: new Date().getTime(),
            type: "connect",
          },
          ...prev,
        ]);
      });

      socket.addEventListener("close", (event) => {
        setMessages((prev) => [
          {
            data: `Disconnected from ${websocketAddress}`,
            ts: new Date().getTime(),
            type: "disconnect",
          },
          ...prev,
        ]);

        websocketRef.current = undefined;
      });
    } else if (websocketRef.current) {
      websocketRef.current.close();
    }

    return () => websocketRef.current?.close();
  }, [connected]);

  const sendMessage = () => {
    if (currentPayload) {
      websocketRef.current?.send(currentPayload);
      setMessages((prev) => [
        {
          data: currentPayload,
          ts: new Date().getTime(),
          type: "message-out",
        },
        ...prev,
      ]);
    }
  };

  const clearMessages = async () => {
    if (!selectedWebsocket) return;

    await toast.promise(
      fetch(
        `http://${getHost()}/api/ws-clear-messages?socket=${encodeURIComponent(
          selectedWebsocket.name
        )}`,
        {
          method: "DELETE",
        }
      ),
      {
        error: "Error clearinging messages",
        loading: "Clearing messages",
        success: "Messages cleared",
      }
    );
  };

  return (
    <AppLayout
      title="Websocket Explorer"
      routePath={"/websockets"}
      secondLevelNav={
        data?.websockets?.length && selectedWebsocket ? (
          <>
            <div className="flex mb-2 items-center justify-between px-2">
              <span className="text-lg">Websockets</span>
            </div>
            <WSTreeView
              initialItem={selectedWebsocket}
              onSelect={(ws) => {
                setselectedWebsocket(ws);
              }}
              websockets={data.websockets}
            />
          </>
        ) : null
      }
    >
      <Loading delay={400} conditionToShow={!loading}>
        {selectedWebsocket ? (
          <div className="flex max-w-6xl flex-col md:pr-8">
            <div className="w-full flex flex-col gap-8">
              <h2 className="text-2xl">{selectedWebsocket?.name}</h2>
              <div>
                <nav
                  className="flex h-10 items-end lg:items-center gap-4"
                  aria-label="Breadcrumb"
                >
                  <div className="flex w-full items-center lg:hidden gap-4">
                    {data?.websockets?.length ? (
                      <Select
                        value={selectedWebsocket.name}
                        onValueChange={(socketName) => {
                          const ws = data?.websockets.find(
                            (ws) => ws.name === socketName
                          );

                          setselectedWebsocket(ws);
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder="Select Message Type" />
                        </SelectTrigger>
                        <SelectContent>
                          {data?.websockets.map((ws) => (
                            <SelectItem key={ws.name} value={ws.name}>
                              {ws.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    ) : null}
                  </div>
                  <div className="hidden lg:flex items-center">
                    <span className="text-lg flex gap-2">
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span
                            data-testid="generated-request-path"
                            className="truncate max-w-xl"
                          >
                            {websocketAddress}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>{websocketAddress}</p>
                        </TooltipContent>
                      </Tooltip>

                      <Tooltip>
                        <TooltipTrigger asChild>
                          <button
                            type="button"
                            onClick={() => {
                              copyToClipboard(websocketAddress);
                              toast.success("Copied Websocket URL");
                            }}
                          >
                            <span className="sr-only">Copy Route URL</span>
                            <ClipboardIcon className="w-5 h-5 text-gray-500" />
                          </button>
                        </TooltipTrigger>
                        <TooltipContent>
                          <p>Copy</p>
                        </TooltipContent>
                      </Tooltip>
                    </span>
                  </div>
                  {tab === "send-messages" && (
                    <div className="ml-auto">
                      <Button
                        onClick={() => setConnected(!connected)}
                        size={"lg"}
                        data-testid="connect-btn"
                        variant={connected ? "destructive" : "default"}
                      >
                        {connected ? "Disconnect" : "Connect"}
                      </Button>
                    </div>
                  )}
                </nav>
              </div>
              <Tabs defaultValue={tab} onValueChange={setTab}>
                <TabsList>
                  <TabsTrigger
                    value="monitor"
                    data-testid="monitor-tab-trigger"
                  >
                    Monitor
                  </TabsTrigger>
                  <TabsTrigger
                    value="send-messages"
                    data-testid="send-messages-tab-trigger"
                  >
                    Send Messages
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="monitor">
                  <Card className="mt-4">
                    <CardHeader className="relative">
                      <CardTitle>Messages</CardTitle>
                      <div className="absolute right-0 top-0 p-6 flex gap-2">
                        <Badge
                          data-testid="connections-status"
                          className="uppercase font-semibold"
                          variant={
                            wsInfo && wsInfo.connectionCount > 0
                              ? "success"
                              : "destructive"
                          }
                        >
                          Connections: {wsInfo?.connectionCount || 0}
                        </Badge>
                      </div>
                      <div className="my-4 pt-4 flex gap-2">
                        <Input
                          placeholder="Search"
                          className="w-4/12"
                          value={monitorMessageFilter}
                          onChange={(evt) =>
                            setMonitorMessageFilter(evt.target.value)
                          }
                        />

                        <Button
                          data-testid="clear-messages-btn"
                          variant="outline"
                          onClick={clearMessages}
                        >
                          <TrashIcon className="mr-2 h-4 w-4" />
                          Clear Messages
                        </Button>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="my-4 max-w-full text-sm">
                        {wsInfo?.messages?.length ? (
                          <ScrollArea
                            className="h-[50vh] w-full px-6"
                            type="always"
                          >
                            {wsInfo.messages
                              .filter((message) => {
                                let pass = true;

                                if (
                                  monitorMessageFilter &&
                                  typeof message.data === "string"
                                ) {
                                  pass = message.data
                                    .toLowerCase()
                                    .includes(
                                      monitorMessageFilter.toLowerCase()
                                    );
                                }

                                return pass;
                              })
                              .map((message, i) => {
                                const shouldBeJSON = /^[{[]/.test(
                                  message.data.trim()
                                );

                                return (
                                  <Accordion type="multiple" key={i}>
                                    <AccordionItem value={message.time}>
                                      <AccordionTrigger className="flex justify-between">
                                        <div>
                                          <MessageIcon
                                            type={
                                              message.data ===
                                              "Binary messages are not currently supported by AWS"
                                                ? "error"
                                                : "message-in"
                                            }
                                          />
                                        </div>
                                        <span
                                          data-testid={`accordion-message-${i}`}
                                          className="px-2 truncate max-w-3xl"
                                        >
                                          {message.data}
                                        </span>
                                        <span className="ml-auto px-2">
                                          {format(
                                            new Date(message.time),
                                            "HH:mm:ss"
                                          )}
                                        </span>
                                      </AccordionTrigger>
                                      <AccordionContent>
                                        {message.data ===
                                        "Binary messages are not currently supported by AWS" ? (
                                          <p>
                                            Binary messages are not currently
                                            supported by AWS. Util this is
                                            supported, use a text-based payload.
                                          </p>
                                        ) : (
                                          <CodeEditor
                                            id="message-viewer"
                                            contentType={
                                              shouldBeJSON
                                                ? "application/json"
                                                : "text/html"
                                            }
                                            readOnly
                                            value={
                                              shouldBeJSON
                                                ? formatJSON(message.data)
                                                : message.data
                                            }
                                            height="208px"
                                            className="h-52"
                                          />
                                        )}
                                      </AccordionContent>
                                    </AccordionItem>
                                  </Accordion>
                                );
                              })}
                          </ScrollArea>
                        ) : (
                          <span className="text-gray-500 text-lg">
                            Send a message to get a response.
                          </span>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>
                <TabsContent value="send-messages" className="space-y-10">
                  <Card className="mt-4">
                    <CardHeader>
                      <CardTitle>Query Params</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="w-full">
                        <FieldRows
                          rows={queryParams}
                          readOnly={connected}
                          testId="query"
                          setRows={(rows) => {
                            setQueryParams(rows);
                          }}
                        />
                      </div>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="flex-row space-y-0 justify-between items-start">
                      <CardTitle>Message</CardTitle>
                    </CardHeader>
                    <CardContent>
                      {payloadType === "text" && (
                        <Textarea
                          placeholder="Enter message"
                          data-testid="message-text-input"
                          value={currentPayload}
                          onChange={(evt) =>
                            setCurrentPayload(evt.target.value)
                          }
                        />
                      )}
                      {["json", "xml", "html"].includes(payloadType) && (
                        <CodeEditor
                          id="message-editor"
                          contentType={
                            {
                              json: "application/json",
                              xml: "application/xml",
                              html: "text/html",
                            }[payloadType] || ""
                          }
                          value={
                            typeof currentPayload === "string"
                              ? currentPayload
                              : ""
                          }
                          height="208px"
                          className="h-52"
                          includeLinters
                          onChange={(value) => {
                            setCurrentPayload(value);
                          }}
                        />
                      )}
                    </CardContent>
                    <CardFooter className="flex gap-2">
                      <Select
                        value={payloadType}
                        onValueChange={setPayloadType}
                      >
                        <SelectTrigger className="w-[150px]">
                          <SelectValue placeholder="Select Message Type" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="text">Text</SelectItem>
                          <SelectItem value="json">JSON</SelectItem>
                          <SelectItem value="xml">XML</SelectItem>
                          <SelectItem value="html">HTML</SelectItem>
                        </SelectContent>
                      </Select>
                      <Button
                        size={"lg"}
                        className="ml-auto"
                        data-testid="send-message-btn"
                        disabled={!currentPayload || !connected}
                        onClick={sendMessage}
                      >
                        Send
                      </Button>
                    </CardFooter>
                  </Card>
                  <Card>
                    <CardHeader className="relative">
                      <CardTitle>Messages</CardTitle>
                      <div className="absolute right-0 top-0 p-6 flex gap-2">
                        <Badge
                          data-testid="connected-status"
                          className="uppercase font-semibold"
                          variant={connected ? "success" : "destructive"}
                        >
                          {connected ? "Connected" : "Disconnected"}
                        </Badge>
                      </div>
                      <div className="my-4 pt-4 flex gap-2">
                        <Input
                          placeholder="Search"
                          className="w-4/12"
                          onChange={(evt) => setMessageFilter(evt.target.value)}
                        />
                        <Select
                          value={messageTypeFilter}
                          onValueChange={setMessageTypeFilter}
                        >
                          <SelectTrigger className="w-[150px]">
                            <SelectValue placeholder="Select" />
                          </SelectTrigger>
                          <SelectContent>
                            <SelectItem value="all">All Messages</SelectItem>
                            <SelectItem value="out">Sent</SelectItem>
                            <SelectItem value="in">Recieved</SelectItem>
                          </SelectContent>
                        </Select>
                        <Button
                          variant="outline"
                          onClick={() => setMessages([])}
                        >
                          <TrashIcon className="mr-2 h-4 w-4" />
                          Clear Messages
                        </Button>
                      </div>
                    </CardHeader>
                    <CardContent>
                      <div className="my-4 max-w-full text-sm">
                        {messages.length ? (
                          <ScrollArea className="h-[30vh] px-6" type="always">
                            {messages
                              .filter((message) => {
                                let pass = false;

                                if (messageTypeFilter === "in") {
                                  pass = message.type === "message-in";
                                } else if (messageTypeFilter === "out") {
                                  pass = message.type === "message-out";
                                } else {
                                  pass = true;
                                }

                                if (
                                  messageFilter &&
                                  typeof message.data === "string"
                                ) {
                                  pass = message.data
                                    .toLowerCase()
                                    .includes(messageFilter.toLowerCase());
                                }

                                return pass;
                              })
                              .map((message, i) => {
                                const shouldBeJSON = /^[{[]/.test(
                                  message.data.trim()
                                );

                                return (
                                  <Accordion type="multiple" key={i}>
                                    <AccordionItem
                                      value={message.ts.toString()}
                                    >
                                      <AccordionTrigger className="flex justify-between">
                                        <div>
                                          <MessageIcon type={message.type} />
                                        </div>
                                        <span
                                          data-testid={`accordion-message-${i}`}
                                          className="px-2 truncate"
                                        >
                                          {message.data}
                                        </span>
                                        <span className="ml-auto px-2">
                                          {format(
                                            new Date(message.ts),
                                            "HH:mm:ss"
                                          )}
                                        </span>
                                      </AccordionTrigger>
                                      <AccordionContent>
                                        <CodeEditor
                                          id="message-viewer"
                                          contentType={
                                            shouldBeJSON
                                              ? "application/json"
                                              : "text/html"
                                          }
                                          readOnly
                                          value={
                                            shouldBeJSON
                                              ? formatJSON(message.data)
                                              : message.data
                                          }
                                          height="208px"
                                          className="h-52"
                                        />
                                      </AccordionContent>
                                    </AccordionItem>
                                  </Accordion>
                                );
                              })}
                          </ScrollArea>
                        ) : (
                          <span className="text-gray-500 text-lg">
                            Send a message to get a response.
                          </span>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </TabsContent>
              </Tabs>
            </div>
          </div>
        ) : (
          <div>
            Please refer to our documentation on{" "}
            <a
              className="underline"
              target="_blank"
              href="https://nitric.io/docs/websockets"
              rel="noreferrer"
            >
              creating Websockets
            </a>{" "}
            as we are unable to find any existing Websockets.
          </div>
        )}
      </Loading>
    </AppLayout>
  );
};

export default WSExplorer;
