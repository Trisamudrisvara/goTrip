# Use the official Go image as the base image
FROM golang:1.23

# Set the working directory in the container
WORKDIR /app

# Copy the application files into the working directory
COPY . /app

# Build the application
RUN go build -tags netgo -ldflags '-s -w' -o app

# Expose port
EXPOSE $API_PORT

# Define the entry point for the container
CMD ["./app"]
