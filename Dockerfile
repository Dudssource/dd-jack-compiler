FROM golang:1.25-alpine
WORKDIR /app
COPY . ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o JackCompiler
RUN chmod +x JackCompiler
CMD ["/app/JackCompiler"]