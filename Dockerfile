FROM node:24-alpine AS build
WORKDIR /app
RUN corepack enable
COPY package.json ./
RUN pnpm install
COPY . .
RUN pnpm build

FROM node:24-alpine
WORKDIR /app
RUN corepack enable
ENV NODE_ENV=production
COPY --from=build /app/package.json ./package.json
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/build ./build
COPY --from=build /app/migrations ./migrations
COPY --from=build /app/scripts ./scripts
EXPOSE 3000
CMD ["node", "build"]
