FROM rsp/openvino-runtime:dev

ADD loss-prevention-service entrypoint.sh /
ADD /res/docker /res/docker

ARG GIT_COMMIT=unspecified
LABEL git_commit=$GIT_COMMIT

WORKDIR /

ENTRYPOINT ["/entrypoint.sh"]
