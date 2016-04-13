FROM docker:1.8.3
MAINTAINER admin@acale.ph

ADD run.sh /usr/bin/docker-agent
RUN chmod +x /usr/bin/docker-agent

ENTRYPOINT docker-agent