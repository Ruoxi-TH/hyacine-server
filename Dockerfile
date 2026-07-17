FROM node:20-alpine

WORKDIR /app

ENV PNPM_HOME=/pnpm \
    PATH=/pnpm:$PATH \
    CI=true

# Install build dependencies for native modules (argon2, etc.)
RUN apk add --no-cache python3 make g++

COPY package.json pnpm-lock.yaml ./
COPY prisma ./prisma

RUN corepack enable && pnpm install --frozen-lockfile

COPY nest-cli.json tsconfig.json tsconfig.build.json ./
COPY src ./src

RUN pnpm prisma:generate \
  && pnpm build \
  && pnpm prune --prod

# Remove build dependencies and cleanup
RUN apk del python3 make g++ \
  && rm -rf /root/.local /root/.cache /tmp/* src nest-cli.json tsconfig.json tsconfig.build.json

ENV NODE_ENV=production

EXPOSE 3000

CMD ["sh", "-c", "pnpm prisma:deploy && node dist/main"]