FROM ubuntu:latest
COPY git-backup /bin/git-backup
ENV DATA_DIR="/data"
CMD ["/bin/git-backup"]
