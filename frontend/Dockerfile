FROM node:24-alpine AS build

WORKDIR /src

RUN corepack enable

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .
ARG VITE_CAMERA_STREAM_URL=http://192.168.31.67/stream
ENV VITE_CAMERA_STREAM_URL=$VITE_CAMERA_STREAM_URL
RUN pnpm build

FROM nginx:1.29-alpine

COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /src/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
