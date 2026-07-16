import { IsString, MinLength } from 'class-validator';

export class NeteaseRecommendationsDto {
  @IsString()
  @MinLength(12)
  cookie!: string;
}
