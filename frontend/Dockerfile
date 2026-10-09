FROM node:22-alpine AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY . .
RUN npm run build && node scripts/csp.mjs

FROM nginxinc/nginx-unprivileged:1.29-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/out /usr/share/nginx/html
COPY --from=build /app/csp-hashes.txt /tmp/csp-hashes.txt
USER root
RUN sed -i "s|__SCRIPT_HASHES__|$(cat /tmp/csp-hashes.txt)|" /etc/nginx/conf.d/default.conf && rm /tmp/csp-hashes.txt
USER 101
EXPOSE 8080
