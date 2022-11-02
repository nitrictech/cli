FROM "node:alpine"

ARG HANDLER
ENV HANDLER=${HANDLER}

COPY package.json *.lock *-lock.json /

RUN yarn import || echo Lockfile already exists

RUN set -ex; yarn install --production --frozen-lockfile --cache-folder /tmp/.cache; rm -rf /tmp/.cache;

COPY . .

ENTRYPOINT node $HANDLER
