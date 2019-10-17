FROM rsp/openvino-runtime:dev

ADD loss-prevention-service entrypoint.sh /
ADD /res /res

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

WORKDIR /

ENTRYPOINT ["/entrypoint.sh"]
