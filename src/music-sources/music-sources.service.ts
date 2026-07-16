import { Injectable, ServiceUnavailableException } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';

interface NeteaseQrResponse { data?: { unikey?: string; qrurl?: string }; code?: number; }
interface NeteaseStatusResponse { code?: number; cookie?: string; message?: string; }
interface NeteaseRecommendResponse { recommend?: Array<{ id?: number; name?: string; picUrl?: string; playcount?: number; trackCount?: number; copywriter?: string }>; code?: number; }
export interface NeteasePlaylist { id: number; name: string; coverUrl: string; playCount: number; trackCount: number; description: string; }

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

  validateBilibiliCookie(cookie: string): { valid: boolean } {
    const names = new Set(cookie.split(';').map((part) => part.trim().split('=')[0]));
    return { valid: names.has('SESSDATA') && names.has('bili_jct') };
  }

  private neteaseBaseUrl(): string {
    const value = this.config.get<string>('NETEASE_API_BASE');
    if (!value) throw new ServiceUnavailableException('NETEASE_API_BASE is not configured');
    return value.replace(/\/$/, '');
  }

  private async request<T>(base: string, path: string, cookie?: string): Promise<T> {
    try {
      const response = await fetch(`${base}${path}`, { headers: { Accept: 'application/json', ...(cookie ? { Cookie: cookie } : {}) } });
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      return await response.json() as T;
    } catch {
      throw new ServiceUnavailableException('Netease QR provider is unavailable');
    }
  }
}