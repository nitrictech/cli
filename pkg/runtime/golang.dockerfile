FROM golang:alpine as build

ARG HANDLER

RUN apk update
RUN apk upgrade
RUN apk add --no-cache git gcc g++ make

WORKDIR /app/

COPY go.mod *.sum ./

RUN go mod download

COPY . .

RUN go build -o /bin/main ./${HANDLER}/...

FROM alpine

COPY --from=build /bin/main /bin/main

RUN chmod +x-rw /bin/main
RUN apk add --no-cache tzdata

ENTRYPOINT ["/bin/main"]