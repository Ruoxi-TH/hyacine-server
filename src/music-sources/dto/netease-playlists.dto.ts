import { IsString, MinLength } from 'class-validator';

export class NeteasePlaylistsDto {
  @IsString()
  @MinLength(12)
  cookie!: string;
}
