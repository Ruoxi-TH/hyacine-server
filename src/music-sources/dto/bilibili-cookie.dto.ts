import { IsString, MinLength } from 'class-validator';

export class BilibiliCookieDto {
  @IsString()
  @MinLength(12)
  cookie!: string;
}