FROM ubuntu:latest
RUN apt update -qy && apt install -qy ca-certificates
COPY git-backup /bin/git-backup
ENV DATA_DIR="/data"
CMD ["/bin/git-backup"]
