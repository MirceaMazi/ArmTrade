import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { map } from 'rxjs/operators';

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
  private apiUrl = 'http://localhost:8080/api';

  constructor(private http: HttpClient) { }

  searchStocks(query: string): Observable<SearchResult[]> {
    return this.http.get<any>(`${this.apiUrl}/search?q=${query}`).pipe(
      map(response => response.quotes || [])
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
}
