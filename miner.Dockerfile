# Minimal Docker image for 0chain miner using prebuilt binary
FROM alpine:3.18

WORKDIR /app

# Copy the prebuilt miner binary from the local build context
COPY code/go/0chain.net/miner/miner/miner /app/miner

# Expose necessary ports (replace with actual ports used by your miner)
EXPOSE 7071 7072 7073

# Set environment variables as needed
ENV ZCHAIN_ENV=production

# Run the miner
ENTRYPOINT ["/app/miner"]
