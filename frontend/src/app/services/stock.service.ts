import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface SearchResult {
  symbol: string;
  shortname: string;
  longname: string;
  exchange: string;
  quoteType: string;
}

export interface AnalysisRequest {
  ticker: string;
  persona?: string;
  whatIf?: string;
}

export interface Annotation {
  date: string;
  description: string;
  type: 'bullish' | 'bearish' | 'info';
}

export interface ArmandAnalysis {
  recommendation: 'BUY' | 'HOLD' | 'SELL';
  reasoning: string[];
  socialSentiment: 'Bullish' | 'Bearish' | 'Neutral';
  annotations: Annotation[];
  confidence?: number;
  targetPrice?: number;
  horizon?: string;
  risks?: string[];
}

export interface ScreenerResult {
  ticker: string;
  name: string;
  reason: string;
}

export interface ScreenerResponse {
  results: ScreenerResult[];
  summary: string;
}

@Injectable({
  providedIn: 'root'
})
export class StockService {
  private apiUrl = environment.apiUrl;

  constructor(private http: HttpClient) { }

  searchStocks(query: string): Observable<SearchResult[]> {
    return this.http.get<any>(`${this.apiUrl}/search?q=${query}`).pipe(
      map(response => (response.quotes || []).filter(
        (item: any) => item.quoteType === 'EQUITY' || item.quoteType === 'ETF'
      ))
    );
  }

  getChart(ticker: string, interval: string = '1d', range: string = '1mo'): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/chart/${ticker}?interval=${interval}&range=${range}`);
  }

  getFundamentals(ticker: string): Observable<any> {
    return this.http.get<any>(`${this.apiUrl}/fundamentals/${ticker}`);
  }

  getArmandAnalysis(request: AnalysisRequest): Observable<ArmandAnalysis> {
    return this.http.post<ArmandAnalysis>(`${this.apiUrl}/armand/analyze`, request);
  }

  screenStocks(query: string): Observable<ScreenerResponse> {
    return this.http.post<ScreenerResponse>(`${this.apiUrl}/armand/screener`, { query });
  }

  getNews(ticker: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.apiUrl}/news/${ticker}`);
  }

  getDividends(ticker: string): Observable<any[]> {
    return this.http.get<any[]>(`${this.apiUrl}/dividends/${ticker}`);
  }

  compareStocks(ticker1: string, ticker2: string): Observable<any> {
    return this.http.post<any>(`${this.apiUrl}/armand/compare`, { ticker1, ticker2 });
  }

  summarizeEarnings(transcript: string, ticker?: string): Observable<any> {
    return this.http.post<any>(`${this.apiUrl}/armand/earnings`, { transcript, ticker });
  }

  saveAnalysis(data: { ticker: string; recommendation: string; reasoning: string[]; persona: string }): Observable<any> {
    return this.http.post(`${this.apiUrl}/analyses`, data);
  }

  getSavedAnalyses(): Observable<any[]> {
    return this.http.get<any[]>(`${this.apiUrl}/analyses`);
  }
}
