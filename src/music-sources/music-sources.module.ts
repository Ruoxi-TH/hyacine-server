import { Module } from '@nestjs/common';
import { MusicSourcesController } from './music-sources.controller';
import { MusicSourcesService } from './music-sources.service';

@Module({
  controllers: [MusicSourcesController],
  providers: [MusicSourcesService],
})
export class MusicSourcesModule {}