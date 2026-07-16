import { Body, Controller, Get, HttpCode, Param, Post } from '@nestjs/common';
import { BilibiliCookieDto } from './dto/bilibili-cookie.dto';
import { NeteaseRecommendationsDto } from './dto/netease-recommendations.dto';
import { MusicSourcesService, type NeteasePlaylist } from './music-sources.service';

@Controller('music-sources')
export class MusicSourcesController {
  constructor(private readonly sources: MusicSourcesService) {}

  @Get('netease/qr')
  createNeteaseQr(): Promise<{ key: string; qrUrl: string }> {
    return this.sources.createNeteaseQr();
  }

  @Get('netease/qr/:key')
  pollNeteaseQr(@Param('key') key: string): Promise<{ status: 'pending' | 'confirmed' | 'expired'; cookie?: string; message?: string }> {
    return this.sources.pollNeteaseQr(key);
  }

  @Post('netease/recommendations')
  @HttpCode(200)
  getNeteaseRecommendations(@Body() dto: NeteaseRecommendationsDto): Promise<NeteasePlaylist[]> {
    return this.sources.getNeteaseRecommendations(dto.cookie);
  }

  @Post('bilibili/validate-cookie')
  @HttpCode(200)
  validateBilibiliCookie(@Body() dto: BilibiliCookieDto): { valid: boolean } {
    return this.sources.validateBilibiliCookie(dto.cookie);
  }
}