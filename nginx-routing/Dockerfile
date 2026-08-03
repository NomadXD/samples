# Using this base image because of: https://github.com/openresty/docker-openresty/issues/124
FROM openresty/openresty:latest

# Add the REDIS connection URL as an env variable in NGINX
# RUN echo -e "env FLY_REDIS_CACHE_URL;\n$(cat /usr/local/openresty/nginx/conf/nginx.conf)" > /usr/local/openresty/nginx/conf/nginx.conf

# Add the configuration file
RUN rm /etc/nginx/conf.d/*
COPY nginx.conf /etc/nginx/conf.d/default.conf
