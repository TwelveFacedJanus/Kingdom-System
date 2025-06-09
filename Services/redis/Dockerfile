FROM redis:7.2-alpine

COPY redis.conf /usr/local/etc/redis/redis.conf

RUN mkdir -p /data && chown redis:redis /data
RUN mkdir -p /var/log/redis && chown redis:redis /var/log/redis
RUN chown redis:redis /usr/local/etc/redis/redis.conf

USER redis
EXPOSE 6379

CMD ["sh", "-c", "redis-server /usr/local/etc/redis/redis.conf --requirepass \"${REDIS_PASSWORD}\""]
