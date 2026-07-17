FROM node:20-alpine AS base
WORKDIR /app
ENV PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH \
    CI=true \
    COREPACK_ENABLE_DOWNLOAD_PROMPT=0
RUN corepack enable

FROM base AS deps
# native build tools for argon2 etc.
RUN apk add --no-cache python3 make g++
COPY package.json pnpm-lock.yaml ./
COPY prisma ./prisma
RUN pnpm install --frozen-lockfile

FROM deps AS build
COPY nest-cli.json tsconfig.json tsconfig.build.json ./
COPY src ./src
RUN pnpm prisma:generate && pnpm build

FROM base AS production
ENV NODE_ENV=production
# openssl needed by Prisma engines on alpine
RUN apk add --no-cache openssl tini \
  && addgroup -S hyacine \
  && adduser -S -G hyacine hyacine
WORKDIR /app

COPY package.json pnpm-lock.yaml ./
COPY prisma ./prisma
# only production deps in final image
RUN pnpm install --frozen-lockfile --prod \
  && pnpm prisma:generate \
  && chown -R hyacine:hyacine /app
COPY --from=build --chown=hyacine:hyacine /app/dist ./dist
USER hyacine
EXPOSE 3000
HEALTHCHECK --interval=15s --timeout=5s --start-period=30s --retries=5 \
  CMD wget -qO- http://127.0.0.1:3000/api/v1/health >/dev/null 2>&1 || exit 1
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["sh", "-c", "pnpm prisma:deploy && node dist/main"]