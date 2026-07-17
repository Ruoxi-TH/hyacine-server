import { IsOptional, IsString, MinLength } from 'class-validator';

export class BilibiliPlayUrlDto {
  @IsString()
  @MinLength(3)
  id!: string;

  @IsOptional()
  @IsString()
  cid?: string;

  @IsOptional()
  @IsString()
  @MinLength(12)
  cookie?: string;
}