import { IsString, MaxLength, MinLength } from 'class-validator';

export class NeteaseCreatePlaylistDto {
  @IsString()
  @MinLength(1)
  @MaxLength(100)
  name!: string;

  @IsString()
  @MinLength(12)
  cookie!: string;
}