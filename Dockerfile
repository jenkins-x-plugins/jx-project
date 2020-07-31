FROM gcr.io/jenkinsxio-labs-private/jxl-base:0.0.55

ENTRYPOINT ["jx-project"]

COPY ./build/linux/jx-project /usr/bin/jx-project