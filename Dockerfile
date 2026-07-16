FROM node:20-bookworm-slim AS base
WORKDIR /app
RUN corepack enable

FROM base AS dependencies
COPY package.json pnpm-lock.yaml ./
RUN corepack pnpm install --frozen-lockfile

FROM dependencies AS build
COPY prisma ./prisma
COPY src ./src
COPY nest-cli.json tsconfig.json tsconfig.build.json ./
RUN corepack pnpm prisma:generate && corepack pnpm build

FROM node:20-bookworm-slim AS production
WORKDIR /app
ENV NODE_ENV=production
RUN corepack enable
COPY package.json pnpm-lock.yaml ./
RUN corepack pnpm install --frozen-lockfile
COPY prisma ./prisma
COPY --from=build /app/dist ./dist
RUN corepack pnpm prisma:generate
EXPOSE 3000
CMD ["sh", "-c", "corepack pnpm prisma:deploy && node dist/main"]