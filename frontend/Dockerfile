FROM node:26-alpine@sha256:6bd9ab43c0ec0190ca7b2d193906cb8b7edf6e7f4afa43ddeccd6452710bffb5 AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build && node scripts/csp.mjs

FROM nginxinc/nginx-unprivileged:1.29-alpine@sha256:0c79d56aee561a1d81c63f00eee5fb5fe29279560cdc55e91425133104c7fbe6
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/out /usr/share/nginx/html
COPY --from=build /app/csp-hashes.txt /tmp/csp-hashes.txt
USER root
# Base image lags Alpine security fixes; patch at build time (trivy gate in CI).
RUN apk upgrade --no-cache && apk del --no-cache curl
RUN sed -i "s|__SCRIPT_HASHES__|$(cat /tmp/csp-hashes.txt)|" /etc/nginx/conf.d/default.conf && rm /tmp/csp-hashes.txt
USER 101
EXPOSE 8080
