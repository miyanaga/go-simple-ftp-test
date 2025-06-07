#!/bin/bash

docker run \
  -e APP_USER_NAME=ftps	\
  -e APP_USER_PASSWD=pass	\
  -e APP_UID=1000	\
  -e APP_GID=1000	\
  -e PASSV_MIN_PORT=60000	\
  -e PASSV_MAX_PORT=60010	\
  -e FORCE_REINIT_CONFIG=false                  `#optional` \
  -e USE_SSL=true                               `#optional` \
  -e APP_UMASK=007                              `#optional` \
  -p 0.0.0.0:21000:21 \
  -p 0.0.0.0:60000-60010:60000-60010 \
  -v $PWD/testdata:/home/vsftpd/data \
  chonjay21/ftps
