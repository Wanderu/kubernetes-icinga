FROM golang:latest AS build
WORKDIR /go/src/github.com/Nexinto/kubernetes-icinga
COPY . .
RUN go get k8s.io/client-go/...
RUN go get -d ./...
RUN CGO_ENABLED=0 GOOS=linux go build -o kubernetes-icinga pkg/controller/*.go

RUN mkdir -p /tmp
RUN chmod -R 1777 /tmp

FROM scratch
COPY --from=build /go/src/github.com/Nexinto/kubernetes-icinga/kubernetes-icinga /kubernetes-icinga
# /tmp directory creation needed for glog (used by client-go) to log. otherwise, we risk
# certain kinds of API errors getting logged into a directory not
# available in a `FROM scratch` Docker container, causing glog to abort
# hard with an exit code > 0.
COPY --from=build /tmp /tmp

CMD ["/kubernetes-icinga"]
