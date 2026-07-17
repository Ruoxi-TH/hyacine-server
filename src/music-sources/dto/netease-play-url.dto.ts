import { IsInt, IsOptional, IsString, Min, MinLength } from 'class-validator';

export class NeteasePlayUrlDto {
  @IsInt()
  @Min(1)
  id!: number;

  @IsOptional()
  @IsString()
  @MinLength(12)
  cookie?: string;

  @IsOptional()
  @IsString()
  level?: string;
}