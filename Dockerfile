FROM gcr.io/jenkinsxio/jx-cli-base:latest

ENTRYPOINT ["jx-project"]

COPY ./build/linux/jx-project /usr/bin/jx-project