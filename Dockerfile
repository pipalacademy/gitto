FROM golang:1.18-alpine
ADD . /app
WORKDIR /app
RUN go build

FROM alpine
RUN apk update && apk add git-daemon
COPY --from=0 /app/gitto /gitto
CMD ["/gitto"]

