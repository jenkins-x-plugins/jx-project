FROM gcr.io/jenkinsxio/jx-cli-base:0.0.21

ENTRYPOINT ["jx-project"]

COPY ./build/linux/jx-project /usr/bin/jx-project