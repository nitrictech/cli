FROM "node:alpine"

ARG HANDLER
ENV HANDLER=${HANDLER}

RUN apk update && \
    apk add --no-cache ca-certificates && \
    update-ca-certificates

COPY . .

RUN yarn import || echo Lockfile already exists

RUN set -ex; yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;

ENTRYPOINT node $HANDLER
