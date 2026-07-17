import { Injectable, ServiceUnavailableException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { createHash } from 'node:crypto';

// Interfaces for Netease API responses
interface NeteaseQrResponse { data?: { unikey?: string; qrurl?: string }; code?: number; }
interface NeteaseStatusResponse { code?: number; cookie?: string; message?: string; }
interface NeteaseRecommendResponse { recommend?: Array<{ id?: number; name?: string; picUrl?: string; playcount?: number; trackCount?: number; copywriter?: string }>; code?: number; }
interface NeteaseAccountResponse { account?: { id?: number }; profile?: { userId?: number }; code?: number; }
interface NeteaseUserPlaylistsResponse { playlist?: Array<{ id?: number; name?: string; coverImgUrl?: string; playCount?: number; trackCount?: number; description?: string }>; code?: number; }
interface NeteaseSearchResponse { result?: { songs?: Array<{ id?: number; name?: string; artists?: Array<{ name?: string }>; ar?: Array<{ name?: string }>; album?: { name?: string; picUrl?: string }; al?: { name?: string; picUrl?: string }; duration?: number; dt?: number }> }; code?: number; }
interface NeteasePlayUrlResponse { data?: Array<{ id?: number; url?: string; br?: number; size?: number; md5?: string; code?: number }>; code?: number; }

// Interfaces for Bilibili API responses
interface BilibiliNavResponse { code?: number; data?: { isLogin?: boolean; wbi_img?: { img_url?: string; sub_url?: string } }; }
interface BilibiliSearchResponse { code?: number; data?: { result?: Array<{ bvid?: string; title?: string; author?: string; pic?: string; duration?: string; type?: string; typename?: string }> }; }
interface BilibiliPlayUrlResponse { code?: number; data?: { durl?: Array<{ url?: string; size?: number; length?: number }>; dash?: { audio?: Array<{ id?: number; baseUrl?: string; backupUrl?: string[] }> } }; }

export interface NeteasePlaylist { id: number; name: string; coverUrl: string; playCount: number; trackCount: number; description: string; }
export interface NeteaseTrack { id: number; title: string; artists: string[]; album: string; coverUrl: string; durationMs: number; source: 'netease'; }
export interface BilibiliTrack { id: string; title: string; artists: string[]; coverUrl: string; duration: string; source: 'bilibili'; }

@Injectable()
export class MusicSourcesService {
  constructor(private readonly config: ConfigService) {}

  async createNeteaseQr(): Promise<{ key: string; qrUrl: string }> {
    const base = this.neteaseBaseUrl();
    const keyResponse = await this.request<NeteaseQrResponse>(base, '/login/qr/key?timestamp=' + Date.now());
    const key = keyResponse.data?.unikey;
    if (!key) throw new ServiceUnavailableException('Netease QR provider returned no session key');
    const qrResponse = await this.request<NeteaseQrResponse>(base, `/login/qr/create?key=${encodeURIComponent(key)}&qrimg=true&timestamp=${Date.now()}`);
    const qrUrl = qrResponse.data?.qrurl;
    if (!qrUrl) throw new ServiceUnavailableException('Netease QR provider returned no QR URL');
    return { key, qrUrl };
  }

  async pollNeteaseQr(key: string): Promise<{ status: 'pending' | 'confirmed' | 'expired'; cookie?: string; message?: string }> {
    const result = await this.request<NeteaseStatusResponse>(this.neteaseBaseUrl(), `/login/qr/check?key=${encodeURIComponent(key)}&timestamp=${Date.now()}`);
    if (result.code === 803 && result.cookie) return { status: 'confirmed', cookie: result.cookie };
    if (result.code === 800) return { status: 'expired', message: result.message };
    return { status: 'pending', message: result.message };
  }

  async getNeteaseRecommendations(cookie: string): Promise<NeteasePlaylist[]> {
    const result = await this.request<NeteaseRecommendResponse>(this.neteaseBaseUrl(), `/recommend/resource?timestamp=${Date.now()}`, cookie);
    return (result.recommend ?? []).flatMap((item) => item.id && item.name && item.picUrl ? [{
      id: item.id,
      name: item.name,
      coverUrl: item.picUrl,
      playCount: item.playcount ?? 0,
      trackCount: item.trackCount ?? 0,
      description: item.copywriter ?? '',
    }] : []);
  }

  async getNeteasePlaylists(cookie: string): Promise<NeteasePlaylist[]> {
    const base = this.neteaseBaseUrl();
    const account = await this.request<NeteaseAccountResponse>(base, `/user/account?timestamp=${Date.now()}`, cookie);
    const userId = account.profile?.userId ?? account.account?.id;
    if (!userId) throw new ServiceUnavailableException('Netease account is unavailable');
    const result = await this.request<NeteaseUserPlaylistsResponse>(base, `/user/playlist?uid=${encodeURIComponent(String(userId))}&timestamp=${Date.now()}`, cookie);
    return (result.playlist ?? []).flatMap((item) => item.id && item.name && item.coverImgUrl ? [{
      id: item.id,
      name: item.name,
      coverUrl: item.coverImgUrl,
      playCount: item.playCount ?? 0,
      trackCount: item.trackCount ?? 0,
      description: item.description ?? '',
    }] : []);
  }

  async searchNetease(keywords: string, limit = 20, cookie?: string): Promise<NeteaseTrack[]> {
    const query = new URLSearchParams({ keywords: keywords.trim(), limit: String(limit), timestamp: String(Date.now()) });
    const result = await this.request<NeteaseSearchResponse>(this.neteaseBaseUrl(), `/cloudsearch?${query.toString()}`, cookie);
    return (result.result?.songs ?? []).flatMap((song) => song.id && song.name ? [{
      id: song.id,
      title: song.name,
      artists: (song.ar ?? song.artists ?? []).flatMap((artist) => artist.name ? [artist.name] : []),
      album: (song.al ?? song.album)?.name ?? '',
      coverUrl: (song.al ?? song.album)?.picUrl ?? '',
      durationMs: song.dt ?? song.duration ?? 0,
      source: 'netease' as const,
    }] : []);
  }

  async getNeteasePlayUrl(id: number, level = 'exhigh', cookie?: string): Promise<{ url: string; br: number }> {
    const base = this.neteaseBaseUrl();
    const levels = [...new Set([level, 'exhigh', 'higher', 'standard'])];
    for (const candidate of levels) {
      const query = new URLSearchParams({ id: String(id), level: candidate, timestamp: String(Date.now()) });
      const result = await this.request<NeteasePlayUrlResponse>(base, `/song/url/v1?${query.toString()}`, cookie);
      const data = result.data?.[0];
      if (data?.url && data.code === 200) return { url: data.url, br: data.br ?? 0 };
    }
    // Fallback to old endpoint
    const fallbackQuery = new URLSearchParams({ id: String(id), timestamp: String(Date.now()) });
    const fallback = await this.request<NeteasePlayUrlResponse>(base, `/song/url?${fallbackQuery.toString()}`, cookie);
    const fallbackData = fallback.data?.[0];
    if (fallbackData?.url && fallbackData.code === 200) {
      return { url: fallbackData.url, br: fallbackData.br ?? 0 };
    }
    throw new ServiceUnavailableException('Failed to get Netease play URL');
  }

  async validateBilibiliCookie(cookie: string): Promise<{ valid: boolean }> {
    const names = new Set(cookie.split(';').map((part) => part.trim().split('=')[0]));
    if (!names.has('SESSDATA') || !names.has('bili_jct')) return { valid: false };
    try {
      const result = await this.bilibiliRequest<BilibiliNavResponse>('/x/web-interface/nav', cookie);
      return { valid: result.code === 0 && result.data?.isLogin === true };
    } catch {
      return { valid: false };
    }
  }

  async searchBilibili(keywords: string, limit = 20, cookie?: string): Promise<BilibiliTrack[]> {
    if (!cookie?.includes('SESSDATA=')) {
      throw new ServiceUnavailableException('请先在音乐源页绑定有效的哔哩哔哩账号');
    }
    const path = await this.signedBilibiliPath('/x/web-interface/wbi/search/type', {
      keyword: keywords.trim(),
      search_type: 'video',
      page: '1',
      page_size: String(Math.min(limit, 20)),
      tids: '3',
    }, cookie);
    const result = await this.bilibiliRequest<BilibiliSearchResponse>(path, cookie);
    if (result.code !== 0) throw new ServiceUnavailableException('Bilibili search is unavailable');
    return (result.data?.result ?? [])
      .filter((item) => item.type === 'video' && (!item.typename || item.typename.includes('音乐')))
      .flatMap((item) => item.bvid && item.title ? [{
        id: item.bvid,
        title: item.title.replace(/<[^>]*>/g, ''),
        artists: item.author ? [item.author] : [],
        coverUrl: item.pic?.startsWith('//') ? `https:${item.pic}` : item.pic ?? '',
        duration: item.duration ?? '',
        source: 'bilibili' as const,
      }] : []);
  }

  async getBilibiliPlayUrl(id: string, cid?: string, cookie?: string): Promise<{ url: string; quality: string; cid: string }> {
    if (!cookie?.includes('SESSDATA=')) {
      throw new ServiceUnavailableException('请先在音乐源页绑定有效的哔哩哔哩账号');
    }
    const bvid = id.trim();
    if (!bvid) throw new ServiceUnavailableException('Bilibili bvid is required');
    const resolvedCid = (cid ?? '').trim() || await this.resolveBilibiliCid(bvid, cookie);
    if (!resolvedCid) throw new ServiceUnavailableException('Failed to resolve Bilibili cid');

    const query = new URLSearchParams({
      bvid,
      cid: resolvedCid,
      qn: '80',
      fnval: '16',
      fourk: '1',
    });
    const result = await this.bilibiliRequest<BilibiliPlayUrlResponse>(`/x/player/playurl?${query.toString()}`, cookie);
    if (result.code !== 0) {
      throw new ServiceUnavailableException(`Bilibili playurl failed: code ${result.code}`);
    }

    const dash = result.data?.dash?.audio ?? [];
    if (dash.length > 0) {
      const audio = [...dash].sort((a, b) => (b.id ?? 0) - (a.id ?? 0))[0];
      const url = audio?.baseUrl || audio?.backupUrl?.[0] || '';
      if (url) return { url, quality: `dash_${audio?.id ?? 0}`, cid: resolvedCid };
    }

    const durl = result.data?.durl?.[0];
    if (durl?.url) {
      return { url: durl.url, quality: 'durl', cid: resolvedCid };
    }
    throw new ServiceUnavailableException('No playable stream found for Bilibili video');
  }

  private async resolveBilibiliCid(bvid: string, cookie?: string): Promise<string> {
    const result = await this.bilibiliRequest<{
      code?: number;
      data?: { cid?: number; pages?: Array<{ cid?: number }> };
    }>(`/x/web-interface/view?bvid=${encodeURIComponent(bvid)}`, cookie);
    if (result.code !== 0) {
      throw new ServiceUnavailableException(`Bilibili view failed: code ${result.code}`);
    }
    const value = result.data?.cid ?? result.data?.pages?.[0]?.cid;
    return value ? String(value) : '';
  }
  private neteaseBaseUrl(): string {
    const value = this.config.get<string>('NETEASE_API_BASE');
    if (!value) throw new ServiceUnavailableException('NETEASE_API_BASE is not configured');
    return value.replace(/\/$/, '');
  }

  private async signedBilibiliPath(path: string, params: Record<string, string>, cookie?: string): Promise<string> {
    const nav = await this.bilibiliRequest<BilibiliNavResponse>('/x/web-interface/nav', cookie);
    const img = nav.data?.wbi_img?.img_url?.split('/').pop()?.split('.')[0] ?? '';
    const sub = nav.data?.wbi_img?.sub_url?.split('/').pop()?.split('.')[0] ?? '';
    if (!img || !sub) throw new ServiceUnavailableException('Bilibili WBI key is unavailable');
    const table = [46,47,18,2,53,8,23,32,15,50,10,31,58,3,45,35,27,43,5,49,33,9,42,19,29,28,14,39,12,38,41,13,37,48,7,16,24,55,40,61,26,17,0,1,60,51,30,4,22,25,54,21,56,59,6,63,57,62,11,36,20,34,44,52];
    const source = img + sub;
    const mixin = table.map((index) => source[index] ?? '').join('').slice(0, 32);
    const signed = new URLSearchParams({ ...params, wts: String(Math.floor(Date.now() / 1000)) });
    const ordered = [...signed.entries()]
      .sort(([a], [b]) => a.localeCompare(b))
      .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value.replace(/[!'()*]/g, ''))}`)
      .join('&');
    const rid = createHash('md5').update(ordered + mixin).digest('hex');
    return `${path}?${ordered}&w_rid=${rid}`;
  }

  private async bilibiliRequest<T>(path: string, cookie?: string): Promise<T> {
    try {
      const response = await fetch(`https://api.bilibili.com${path}`, {
        headers: {
          Accept: 'application/json',
          Referer: 'https://www.bilibili.com/',
          'User-Agent': 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36',
          ...(cookie ? { Cookie: cookie } : {}),
        },
        signal: AbortSignal.timeout(10_000),
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      return await response.json() as T;
    } catch (err: any) {
      throw new ServiceUnavailableException(err?.message || 'Bilibili provider is unavailable');
    }
  }

  private async request<T>(base: string, path: string, cookie?: string): Promise<T> {
    try {
      const response = await fetch(`${base}${path}`, {
        headers: {
          Accept: 'application/json',
          ...(cookie ? { Cookie: cookie } : {}),
        },
        signal: AbortSignal.timeout(10_000),
      });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      return await response.json() as T;
    } catch (err: any) {
      throw new ServiceUnavailableException(err?.message || 'Netease provider is unavailable');
    }
  }
}