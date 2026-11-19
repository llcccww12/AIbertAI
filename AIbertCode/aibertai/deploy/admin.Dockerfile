FROM node:22.23.3-bookworm-slim AS build
WORKDIR /app
RUN npm install -g pnpm@10.12.4
COPY admin/package.json admin/pnpm-lock.yaml admin/pnpm-workspace.yaml ./
COPY admin/patches ./patches
RUN pnpm install --frozen-lockfile
COPY admin/ ./
RUN pnpm build

FROM node:22.23.3-bookworm-slim
WORKDIR /app
COPY --from=build /app/dist ./dist
COPY admin/scripts/serve-admin.mjs ./scripts/
USER node
CMD ["node", "scripts/serve-admin.mjs"]
