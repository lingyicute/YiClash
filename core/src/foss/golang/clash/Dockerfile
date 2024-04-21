FROM alpine:latest as builder
ARG TARGETPLATFORM
RUN echo "I'm building for $TARGETPLATFORM"

RUN apk add --no-cache gzip && \
    mkdir /yiclashcore-config && \
    wget -O /yiclashcore-config/geoip.metadb https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.metadb && \
    wget -O /yiclashcore-config/geosite.dat https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geosite.dat && \
    wget -O /yiclashcore-config/geoip.dat https://fastly.jsdelivr.net/gh/MetaCubeX/meta-rules-dat@release/geoip.dat

COPY docker/file-name.sh /yiclashcore/file-name.sh
WORKDIR /yiclashcore
COPY bin/ bin/
RUN FILE_NAME=`sh file-name.sh` && echo $FILE_NAME && \
    FILE_NAME=`ls bin/ | egrep "$FILE_NAME.gz"|awk NR==1` && echo $FILE_NAME && \
    mv bin/$FILE_NAME yiclashcore.gz && gzip -d yiclashcore.gz && echo "$FILE_NAME" > /yiclashcore-config/test
FROM alpine:latest
LABEL org.opencontainers.image.source="https://github.com/lingyicute/YiClashCore"

RUN apk add --no-cache ca-certificates tzdata iptables

VOLUME ["/root/.config/yiclashcore/"]

COPY --from=builder /yiclashcore-config/ /root/.config/yiclashcore/
COPY --from=builder /yiclashcore/yiclashcore /yiclashcore
RUN chmod +x /yiclashcore
ENTRYPOINT [ "/yiclashcore" ]
