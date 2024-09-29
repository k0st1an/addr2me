FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN apk --no-cache add make
RUN make build

FROM alpine:3.20
RUN apk --no-cache add bash
COPY --from=builder /app/addr2me /usr/local/sbin/addr2me
USER nobody
EXPOSE 7007
ENTRYPOINT ["addr2me"]
