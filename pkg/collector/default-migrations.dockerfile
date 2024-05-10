# Default migrations dockerfile
FROM migrate/migrate

ENV DB_URL=""
ENV NITRIC_DB_NAME=""

ARG MIGRATIONS_PATH

COPY ${MIGRATIONS_PATH} /migrations

ENTRYPOINT ["sh", "-c", "migrate -path=/migrations -database $DB_URL up"]
