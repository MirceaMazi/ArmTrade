import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export interface CompanyStat {
  symbol: string;
  name: string;
  price: number;
  dayChange: number;
  weekChange: number;
  marketCap: number;
}

export interface SectorNewsItem {
  title: string;
  source: string;
  published: number;
  url: string;
}

export interface SectorPreview {
  slug: string;
  name: string;
  avgChange: number;
  sentiment: 'up' | 'down' | 'flat';
  companies: CompanyStat[];
}

export interface SectorDetail {
  slug: string;
  name: string;
  avgChange: number;
  sentiment: 'up' | 'down' | 'flat';
  companies: CompanyStat[];
  news: SectorNewsItem[];
}

export interface SectorSummary {
  summary: string;
  sentiment: 'Bullish' | 'Bearish' | 'Neutral';
}

export interface SectorSummaryRequest {
  sector: string;
  movers: string[];
  headlines: string[];
}

export interface IpoItem {
  company: string;
  ticker: string;
  date: string;
  exchange: string;
  priceFrom: number;
  priceTo: number;
  priceRange: string;
}

export interface EarningsItem {
  company: string;
  ticker: string;
  date: string;
  time: string;
  epsEstimate: number;
  hasEps: boolean;
  epsForward: boolean;
}

@Injectable({
  providedIn: 'root'
})
export class MarketService {
  private apiUrl = environment.apiUrl;

  constructor(private http: HttpClient) { }

  getSectorsPreview(): Observable<SectorPreview[]> {
    return this.http.get<SectorPreview[]>(`${this.apiUrl}/market/sectors-preview`);
  }

  getSectorDetails(slug: string): Observable<SectorDetail> {
    return this.http.get<SectorDetail>(`${this.apiUrl}/market/sector-details/${slug}`);
  }

  getSectorSummary(request: SectorSummaryRequest): Observable<SectorSummary> {
    return this.http.post<SectorSummary>(`${this.apiUrl}/armand/sector-summary`, request);
  }

  getIpos(): Observable<IpoItem[]> {
    return this.http.get<IpoItem[]>(`${this.apiUrl}/market/ipos`);
  }

  getEarningsCalendar(): Observable<EarningsItem[]> {
    return this.http.get<EarningsItem[]>(`${this.apiUrl}/market/earnings`);
  }
}
