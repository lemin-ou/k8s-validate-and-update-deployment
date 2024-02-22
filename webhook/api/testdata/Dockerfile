FROM --platform=amd64 nginx:alpine

COPY index.html /usr/share/nginx/html
#copying nginx config from local to image
COPY /nginx.conf /etc/nginx/conf.d/default.conf