FROM ubuntu:latest
COPY vxdb /bin/git-backup
ENV DATA_DIR="/data"
CMD ["/bin/git-backup"]