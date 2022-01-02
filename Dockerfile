FROM python:3-alpine

COPY src /app

RUN apk add --update-cache --no-cache bash git && \
    pip3 install --compile --no-cache-dir -r /app/requirements.txt

CMD ["/app/run.sh"]
