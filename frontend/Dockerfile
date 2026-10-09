FROM node:22-alpine@sha256:0a7108bf6c7bf5de370ffb1a3ed6be93d405b43ff159f681a8d18c0e2bc2e402 AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build && node scripts/csp.mjs

FROM nginxinc/nginx-unprivileged:1.31-alpine@sha256:b9241c6e7b8e9a862f129d8d4199ab64b10390949a78bdd5603379b32c844083
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/out /usr/share/nginx/html
COPY --from=build /app/csp-hashes.txt /tmp/csp-hashes.txt
USER root
RUN sed -i "s|__SCRIPT_HASHES__|$(cat /tmp/csp-hashes.txt)|" /etc/nginx/conf.d/default.conf && rm /tmp/csp-hashes.txt
USER 101
EXPOSE 8080
