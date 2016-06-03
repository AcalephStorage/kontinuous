FROM debian:jessie
MAINTAINER admin@acale.ph

ENV KUBERNETES_VERSION 1.2.4
ENV MC_VERSION release

# install curl and git
# download kubectl and mc
RUN apt-get update && \
    apt-get install -y curl git && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    curl -O https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl && \
    mv kubectl /usr/bin && \
    chmod +x /usr/bin/kubectl && \
    curl -O https://dl.minio.io/client/mc/${MC_VERSION}/linux-amd64/mc && \
    mv mc /usr/bin && \
    chmod +x /usr/bin/mc

# copy kubeconfig
ADD kube-config.yml /root/.kube/config

# copy the script
ADD run.sh /usr/bin/kontinuous-agent
RUN chmod +x /usr/bin/kontinuous-agent

ENTRYPOINT kontinuous-agent
