FROM golang:1.19-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG LDFLAGS=""
RUN CGO_ENABLED=0 go build -trimpath -ldflags "$LDFLAGS" -o /out/calendar ./cmd/calendar

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /out/calendar /usr/local/bin/calendar
COPY configs/config.toml /etc/calendar/config.toml

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/calendar"]
CMD ["-config", "/etc/calendar/config.toml"]
