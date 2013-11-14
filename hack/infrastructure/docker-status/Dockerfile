# VERSION:        0.1.5
# AUTHOR:         Daniel Mizyrycki <daniel@docker.com>
# DESCRIPTION:    Build docker-status container
# URLS:           stashboard page:       http://127.0.0.1:8080
#                 stashboard admin page: http://127.0.0.1:8080/admin
#
# TO_BUILD:   docker build -t docker-status .
# USAGE:
#   # Download docker-status and data Dockerfiles
#   wget https://raw.github.com/dotcloud/docker/docker-status/hack/infrastructure/docker-status/Dockerfile
#   wget http://raw.github.com/dotcloud/docker/master/contrib/desktop-integration/data/Dockerfile
#
#   # Local ephemeral run  (data not saved after server container exits)
#   docker run -u 1000 -volumes-from docker-status-data -p 127.0.0.1:8080:8080 -p 127.0.0.1:8000:8000 docker-status
#
#   # remote access ephemeral run (server container process remote queries)
#   docker run -u 1000 -p 8080:8080 -p 8000:8000 docker-status
#
#   # stateful remote access (data is kept on a data container)
#   docker run -name docker-status-data data true   # Create data container
#   docker run -u 1000 -volumes-from docker-status-data -p 8080:8080 -p 8000:8000 docker-status

DOCKER-VERSION 0.6.6

# Base docker image
FROM ubuntu:12.04
MAINTAINER Daniel Mizyrycki <daniel@docker.com>

RUN echo 'deb http://archive.ubuntu.com/ubuntu precise main universe' > \
      /etc/apt/sources.list
RUN apt-get update
RUN apt-get install -y less wget bzip2 unzip python-pip ca-certificates

# Install stashboard and its dependencies
RUN mkdir /application; cd /application; \
      wget -q http://googleappengine.googlecode.com/files/google_appengine_1.8.7.zip; \
    unzip -q /application/google_appengine_1.8.7.zip; rm google_appengine_1.8.7.zip
RUN cd /application; wget -q -O - http://github.com/twilio/stashboard/tarball/master | \
    tar -zx --transform 's/[^\/]*/stashboard/'

# create sysadmin account
RUN useradd -m -d /data sysadmin

# Define default server command
CMD ["/bin/sh", "-c", "/application/google_appengine/dev_appserver.py --skip_sdk_update_check \
      --host 0.0.0.0 --port 8080 --admin_host 0.0.0.0 --datastore_path=/data/docker-status.db \
      /application/stashboard/stashboard/app.yaml"]
