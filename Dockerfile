FROM alpine

ARG BUILDARCH
WORKDIR /app
RUN apk add --no-cache tzdata
COPY ./${BUILDARCH}/release /app/
VOLUME /app/data

EXPOSE 21114
HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -q -T 3 -O /dev/null http://127.0.0.1:21114/health/ready || exit 1
CMD ["./apimain"]
