## Node build stage for webapp
FROM node:18-alpine AS webapp-builder

WORKDIR /webapp

COPY examples/todomailer/webapp/ ./

RUN npm install --legacy-peer-deps

RUN npm run build

## Go build stage
FROM golang:1.25.3 AS go-builder

WORKDIR /build

# Copy the parent symbiont module to /build
COPY . .

WORKDIR /build/examples/todomailer

RUN go mod download

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./cmd/todomailer/main.go

## Minimal runtime image
FROM scratch

COPY --from=go-builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=go-builder /build/examples/todomailer/main .

# Copy webapp static files
COPY --from=webapp-builder /webapp/dist ./webapp/dist

CMD ["/main"]