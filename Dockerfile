FROM python:3-slim

COPY src /app

RUN apt update -qy && apt install -qy bash git openssh-client && \
    pip3 install --compile --no-cache-dir -r /app/requirements.txt

CMD ["/app/run.sh"]
