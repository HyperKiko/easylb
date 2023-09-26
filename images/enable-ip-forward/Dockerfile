FROM alpine
WORKDIR /
COPY images/enable-ip-forward/entrypoint.sh /entrypoint.sh
RUN apk update && apk add iptables && \
    chmod +x /entrypoint.sh
ENTRYPOINT [ "/entrypoint.sh" ]