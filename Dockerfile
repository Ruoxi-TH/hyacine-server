ARG NODE_IMAGE=node:20-alpine
FROM ${NODE_IMAGE} AS base
WORKDIR /app
ENV PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH \
    CI=true \
    COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable

FROM base AS deps
RUN apk add --no-cache python3 make g++ openssl
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY prisma ./prisma
RUN pnpm install --frozen-lockfile

FROM deps AS build
COPY nest-cli.json tsconfig.json tsconfig.build.json ./
COPY src ./src
RUN pnpm exec prisma generate && pnpm build

FROM base AS production
ENV NODE_ENV=production
RUN apk add --no-cache openssl tini \
  && addgroup -S hyacine \
  && adduser -S -G hyacine hyacine
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY prisma ./prisma
RUN pnpm install --frozen-lockfile --prod \
  && pnpm rebuild argon2 @prisma/client prisma \
  && pnpm exec prisma generate \
  && chown -R hyacine:hyacine /app
COPY --from=build --chown=hyacine:hyacine /app/dist ./dist
USER hyacine
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=5 \
  CMD wget -qO- http://127.0.0.1:3000/api/v1/health >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["sh", "-c", "pnpm exec prisma migrate deploy && node dist/main"]
