import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import * as Joi from 'joi';
import { AuthModule } from './auth/auth.module';
import { HealthModule } from './health/health.module';
import { PrismaModule } from './prisma/prisma.module';
import { MusicSourcesModule } from './music-sources/music-sources.module';
import { UsersModule } from './users/users.module';

@Module({
  imports: [
    ConfigModule.forRoot({
      isGlobal: true,
      validationSchema: Joi.object({
        DATABASE_URL: Joi.string().pattern(/^(postgresql|file):/).required(),
        REDIS_URL: Joi.string().uri({ scheme: ['redis'] }).required(),
        PORT: Joi.number().port().default(3000),
        CORS_ORIGIN: Joi.string().required(),
        NETEASE_API_BASE: Joi.string().uri().optional(),
        JWT_ACCESS_SECRET: Joi.string().min(32).required(),
        JWT_REFRESH_SECRET: Joi.string().min(32).required(),
        JWT_ACCESS_TTL: Joi.string().default('15m'),
        JWT_REFRESH_TTL: Joi.string().default('30d'),
      }),
    }),
    PrismaModule,
    HealthModule,
    MusicSourcesModule,
    AuthModule,
    UsersModule,
  ],
})
export class AppModule {}
