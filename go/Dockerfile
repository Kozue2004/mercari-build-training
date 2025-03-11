#STEP7-6
FROM golang:1.24-alpine 

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

RUN apk add --no-cache sqlite-dev gcc musl-dev

COPY . .

RUN CGO_ENABLED=1 go build -o myapp cmd/api/main.go

RUN addgroup -S mercari && adduser -S trainee -G mercari

RUN chown -R trainee:mercari db images

USER trainee
EXPOSE 9000
CMD ["./myapp"]
