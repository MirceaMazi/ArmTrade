import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import { environment } from '../../environments/environment';

export interface WatchlistItem {
  id: string;
  ticker: string;
  price: number;
  change: number;
  buyPrice?: number;
  quantity?: number;
  buyDate?: string;
}

@Injectable({
  providedIn: 'root'
})
export class WatchlistService {
  private apiUrl = `${environment.apiUrl}/watchlist`;

  constructor(private http: HttpClient) {}

  getWatchlist(): Observable<WatchlistItem[]> {
    return this.http.get<WatchlistItem[]>(this.apiUrl);
  }

  addToWatchlist(ticker: string): Observable<any> {
    return this.http.post(this.apiUrl, { ticker });
  }

  removeFromWatchlist(id: string): Observable<any> {
    return this.http.delete(`${this.apiUrl}/${id}`);
  }

  updatePortfolio(id: string, data: { buyPrice?: number; quantity?: number; buyDate?: string }): Observable<any> {
    return this.http.put(`${this.apiUrl}/${id}/portfolio`, data);
  }
}
