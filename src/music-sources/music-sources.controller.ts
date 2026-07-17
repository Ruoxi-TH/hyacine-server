import { Body, Controller, Get, HttpCode, Param, Post } from '@nestjs/common';
import { BilibiliCookieDto } from './dto/bilibili-cookie.dto';
import { BilibiliSearchDto } from './dto/bilibili-search.dto';
import { BilibiliPlayUrlDto } from './dto/bilibili-play-url.dto';
import { NeteaseRecommendationsDto } from './dto/netease-recommendations.dto';
import { NeteasePlaylistsDto } from './dto/netease-playlists.dto';
import { NeteaseSearchDto } from './dto/netease-search.dto';
import { NeteasePlayUrlDto } from './dto/netease-play-url.dto';
import { MusicSourcesService, type BilibiliTrack, type NeteasePlaylist, type NeteaseTrack } from './music-sources.service';

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

  @Post('netease/playlists')
  @HttpCode(200)
  getNeteasePlaylists(@Body() dto: NeteasePlaylistsDto): Promise<NeteasePlaylist[]> {
    return this.sources.getNeteasePlaylists(dto.cookie);
  }

  @Post('netease/search')
  @HttpCode(200)
  searchNetease(@Body() dto: NeteaseSearchDto): Promise<NeteaseTrack[]> {
    return this.sources.searchNetease(dto.keywords, dto.limit, dto.cookie);
  }

  @Post('netease/play-url')
  @HttpCode(200)
  getNeteasePlayUrl(@Body() dto: NeteasePlayUrlDto): Promise<{ url: string; br: number }> {
    return this.sources.getNeteasePlayUrl(dto.id, dto.level, dto.cookie);
  }

  @Post('bilibili/validate-cookie')
  @HttpCode(200)
  validateBilibiliCookie(@Body() dto: BilibiliCookieDto): Promise<{ valid: boolean }> {
    return this.sources.validateBilibiliCookie(dto.cookie);
  }

  @Post('bilibili/search')
  @HttpCode(200)
  searchBilibili(@Body() dto: BilibiliSearchDto): Promise<BilibiliTrack[]> {
    return this.sources.searchBilibili(dto.keywords, dto.limit, dto.cookie);
  }

  @Post('bilibili/play-url')
  @HttpCode(200)
  getBilibiliPlayUrl(@Body() dto: BilibiliPlayUrlDto): Promise<{ url: string; quality: string; cid: string }> {
    return this.sources.getBilibiliPlayUrl(dto.id, dto.cid, dto.cookie);
  }
}