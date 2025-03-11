FROM alpine

RUN apk add --no-cache go

RUN addgroup -S mercari && adduser -S trainee -G mercari
# RUN chown -R trainee:mercari /path/to/db

USER trainee

CMD ["go", "version"]

FROM golang:1.23.6-alpine

WORKDIR /app

COPY go.mod go.sum ./
COPY . .

RUN go mod download

RUN go build -o app .

CMD ["/app/app"]