FROM redis:7.2-alpine

# Add security configurations
COPY redis.conf /usr/local/etc/redis/redis.conf

# Create directory for Redis data
RUN mkdir -p /data && chown redis:redis /data

# Create directory for Redis logs
RUN mkdir -p /var/log/redis && chown redis:redis /var/log/redis

# Set proper permissions
RUN chown redis:redis /usr/local/etc/redis/redis.conf

# Switch to non-root user
USER redis

# Expose Redis port
EXPOSE 6379

# Start Redis with custom config and password from environment variable at runtime
CMD ["sh", "-c", "redis-server /usr/local/etc/redis/redis.conf --requirepass \"${REDIS_PASSWORD}\""]
