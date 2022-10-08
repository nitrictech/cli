FROM python:3.7-slim

ARG HANDLER

ENV HANDLER=${HANDLER}

RUN pip install --upgrade pip

COPY requirements.txt requirements.txt

RUN pip install --no-cache-dir -r requirements.txt

COPY . .

ENTRYPOINT python $HANDLER