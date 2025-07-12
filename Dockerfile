FROM golang:1.24

# Set the working directory inside the container
WORKDIR /app

# Update CA certificates and install git
RUN apt-get update && apt-get install -y ca-certificates git && rm -rf /var/lib/apt/lists/*

# Update CA certificates to fix TLS issues
RUN update-ca-certificates

# Set Go proxy and sum database (optional, but can help with some issues)
ENV GOPROXY=https://proxy.golang.org,direct
ENV GOSUMDB=sum.golang.org

# Copy the Go modules manifests
COPY go.mod go.sum ./
# Download the Go modules
RUN go mod download

# Copy the source code into the container
COPY . .

# Compile the Go application
# CGO_ENABLED=0 disables cgo, GOOS=linux sets the target OS to Linux
# The output binary will be named gin-pizza-api
RUN CGO_ENABLED=0 GOOS=linux go build -o /gin-pizza-api cmd/main.go

EXPOSE 8080

CMD ["/gin-pizza-api"]