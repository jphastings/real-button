FROM golang:1.16-alpine as build

WORKDIR /go/src/github.com/jphastings/real-button
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

FROM alpine
COPY --from=build /go/bin/real-button /bin/real-button
COPY config.yaml /config/config.yaml

ENTRYPOINT ["real-button"]
CMD ["--config", "/config/config.yaml"]
