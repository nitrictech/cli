# syntax=docker/dockerfile:1
FROM openjdk:22-slim

ARG HANDLER

COPY $HANDLER /usr/app/app.jar

CMD ["java", "-jar", "/usr/app/app.jar"]

