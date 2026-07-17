FROM node:20-bookworm-slim

WORKDIR /app

ENV PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH \
    CI=true \
    NODE_OPTIONS=--max-old-space-size=512

RUN apt-get update \
  && apt-get install -y --no-install-recommends openssl ca-certificates \
  && rm -rf /var/lib/apt/lists/* \
  && npm install -g pnpm@10.15.0

COPY package.json pnpm-lock.yaml ./
COPY prisma ./prisma

# Install all deps (including build tools), then prune for production.
RUN pnpm install --frozen-lockfile

COPY nest-cli.json tsconfig.json tsconfig.build.json ./
COPY src ./src

RUN pnpm prisma:generate \
  && pnpm build \
  && pnpm prune --prod \
  && rm -rf /root/.local /root/.cache /tmp/* src nest-cli.json tsconfig.json tsconfig.build.json

ENV NODE_ENV=production

EXPOSE 3000

CMD ["sh", "-c", "pnpm prisma:deploy && node dist/main"]