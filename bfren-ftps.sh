#!/bin/bash

docker run \
  -e BF_FTPS_EXTERNAL_IP=127.0.0.1 \
  -e BF_FTPS_VSFTPD_USER=ftps \
  -e BF_FTPS_VSFTPD_PASS=pass \
  -e BF_FTPS_VSFTPD_UID=1000 \
  -e BF_FTPS_VSFTPD_MIN_PORT=60000 \
  -e BF_FTPS_VSFTPD_MAX_PORT=60010 \
  -e BF_FTPS_VSFTPD_ENABLE_DEBUG_LOG=1 \
  -p 0.0.0.0:21000:21 \
  -p 0.0.0.0:60000-60010:60000-60010 \
  -v $PWD/testdata:/files \
  bfren/ftps
