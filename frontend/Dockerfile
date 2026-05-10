# Multi-stage frontend build.
#
# Stage 1 (deps):    install node_modules with cached layer.
# Stage 2 (dev):     used by docker-compose.override for hot reload.
# Stage 3 (build):   produce static SPA assets.
# Stage 4 (runtime): nginx serving the SPA + reverse-proxying /ws and /api
#                    to the backend container so the browser only ever
#                    talks to one origin.

FROM node:20-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci

FROM deps AS dev
WORKDIR /app
EXPOSE 5173
# Source is bind-mounted at runtime by docker-compose.override.yml.

FROM deps AS build
WORKDIR /app
COPY . .
RUN npm run build

FROM nginx:1.27-alpine AS runtime
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
