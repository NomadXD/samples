# Use an official Golang runtime as a parent image
FROM golang:1.18

# Set the working directory to /go/src/app
WORKDIR /go/src/app

# Copy the current directory contents into the container at /go/src/app
COPY . .

# Download and install any required dependencies
RUN go mod download

# Build the Go application
RUN go build -o main .

# Expose port 8080 for the application
EXPOSE 50051
EXPOSE 8080

# Define the command to run the application when the container starts
CMD ["./main"]