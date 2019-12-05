FROM rsp/gocv-openvino-builder:dev

ADD loss-prevention-service entrypoint.sh /
ADD /res /res

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

WORKDIR /

ENTRYPOINT ["/entrypoint.sh"]
