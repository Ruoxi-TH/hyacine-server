import { IsInt, IsOptional, IsString, Max, Min, MinLength } from 'class-validator';

export class NeteaseSearchDto {
  @IsString()
  @MinLength(1)
  keywords!: string;

  @IsOptional()
  @IsInt()
  @Min(1)
  @Max(50)
  limit?: number;

  @IsOptional()
  @IsString()
  @MinLength(12)
  cookie?: string;
}
