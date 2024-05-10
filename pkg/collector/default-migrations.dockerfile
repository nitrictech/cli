# Default migrations dockerfile
FROM migrate/migrate

ENV DB_URL=""
ENV NITRIC_DB_NAME=""

COPY ./migrations /migrations

ENTRYPOINT ["sh", "-c", "migrate -path=/migrations/$NITRIC_DB_NAME -database $DB_URL up"]
