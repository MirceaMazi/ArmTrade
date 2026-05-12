import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';

export interface WatchlistItem {
  ticker: string;
  price: number;
  change: number;
}

@Injectable({
  providedIn: 'root'
})
export class WatchlistService {
  private apiUrl = 'http://localhost:8080/api/watchlist';

  constructor(private http: HttpClient) {}

  getWatchlist(): Observable<WatchlistItem[]> {
    return this.http.get<WatchlistItem[]>(this.apiUrl);
  }

  addToWatchlist(ticker: string): Observable<any> {
    return this.http.post(this.apiUrl, { ticker });
  }

  removeFromWatchlist(ticker: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${ticker}`);
  }
}
