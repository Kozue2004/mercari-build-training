FROM alpine

RUN apk add --no-cache go

RUN addgroup -S mercari && adduser -S trainee -G mercari
# RUN chown -R trainee:mercari /path/to/db

USER trainee

CMD ["go", "version"]
