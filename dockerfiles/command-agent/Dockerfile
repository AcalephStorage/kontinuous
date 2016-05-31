FROM debian:jessie
MAINTAINER admin@acale.ph

ENV KUBERNETES_VERSION 1.2.4

# install curl
# download kubectl
RUN apt-get update && \
    apt-get install -y curl && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* && \
    curl -O https://storage.googleapis.com/kubernetes-release/release/v${KUBERNETES_VERSION}/bin/linux/amd64/kubectl && \
    mv kubectl /usr/bin && \
    chmod +x /usr/bin/kubectl

ADD kube-config.yml /root/.kube/config
ADD pod_template.yml /root/pod_template.yml
ADD run.sh /usr/bin/command-agent
RUN chmod +x /usr/bin/command-agent

ENTRYPOINT command-agent