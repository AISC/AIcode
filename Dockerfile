FROM golang:1.21 AS build

WORKDIR /src
# enable modules caching in separate layer
COPY go.mod go.sum ./
RUN go mod download
COPY . ./

RUN make binary

FROM debian:12.4-slim

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates; \
    apt-get clean; \
    rm -rf /var/lib/apt/lists/*; \
    groupadd -r aisc --gid 999; \
    useradd -r -g aisc --uid 999 --no-log-init -m aisc;

# make sure mounted volumes have correct permissions
RUN mkdir -p /home/aisc/.aisc && chown 999:999 /home/aisc/.aisc

COPY --from=build /src/dist/aisc /usr/local/bin/aisc

EXPOSE 1633 1634 1635
USER aisc
WORKDIR /home/aisc
VOLUME /home/aisc/.aisc

ENTRYPOINT ["aisc"]
