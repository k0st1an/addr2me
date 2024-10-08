FROM golang:1.23-alpine AS builder

LABEL maintainer="Konstantin Kruglov <kruglovk@gmail.com>"
LABEL repository="github.com/k0st1an/addr2me"

WORKDIR /app
COPY . .
RUN apk --no-cache add make
RUN make build

FROM alpine:3.20
RUN apk --no-cache add bash
COPY --from=builder /app/addr2me /usr/local/sbin/addr2me
COPY --from=builder /app/LICENSE /
COPY --from=builder /app/README.md /
USER nobody
EXPOSE 7007
ENTRYPOINT ["addr2me"]
