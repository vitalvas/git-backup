#!/usr/bin/env bash

export DATA_DIR=/data

if [ ! -f "${DATA_DIR}/id_rsa" ]; then
  ssh-keygen -t rsa -b 4096 -C "git-backup" -f ${DATA_DIR}/id_rsa -q -N ""
fi

echo -n "SSH Key for backup: "
cat ${DATA_DIR}/id_rsa.pub

ssh-keyscan -H github.com >> /etc/ssh/ssh_known_hosts 2>/dev/null

git config --global core.sshCommand "ssh -i ${DATA_DIR}/id_rsa -F /dev/null"

while true; do
  python /app/git-backup.py
  sleep 3600
done
