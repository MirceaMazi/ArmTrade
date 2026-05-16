import { Injectable, OnDestroy } from '@angular/core';
import { Subject, Observable } from 'rxjs';
import { filter, map } from 'rxjs/operators';
import { environment } from '../../environments/environment';

export interface PriceUpdate {
  ticker: string;
  price: number;
  change: number;
  changePct: number;
  prevClose: number;
}

@Injectable({
  providedIn: 'root'
})
export class PriceWsService implements OnDestroy {
  private ws: WebSocket | null = null;
  private updates$ = new Subject<PriceUpdate[]>();
  private reconnectTimer: any;
  private subscribedTickers = new Set<string>();

  constructor() {
    this.connect();
  }

  ngOnDestroy(): void {
    this.disconnect();
  }

  private getWsUrl(): string {
    const apiUrl = environment.apiUrl;
    // Convert http(s) URL to ws(s)
    if (apiUrl.startsWith('http')) {
      const wsUrl = apiUrl.replace(/^http/, 'ws');
      return `${wsUrl}/ws/prices`;
    }
    // Relative URL (production) — use current host
    const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${proto}//${location.host}${apiUrl}/ws/prices`;
  }

  private connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN) return;

    this.ws = new WebSocket(this.getWsUrl());

    this.ws.onopen = () => {
      // Re-subscribe to any tickers we were tracking
      if (this.subscribedTickers.size > 0) {
        this.sendSubscribe(Array.from(this.subscribedTickers));
      }
    };

    this.ws.onmessage = (event) => {
      try {
        const updates: PriceUpdate[] = JSON.parse(event.data);
        this.updates$.next(updates);
      } catch {}
    };

    this.ws.onclose = () => {
      // Auto-reconnect after 3 seconds
      this.reconnectTimer = setTimeout(() => this.connect(), 3000);
    };

    this.ws.onerror = () => {
      this.ws?.close();
    };
  }

  private disconnect(): void {
    clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.ws = null;
  }

  private sendSubscribe(tickers: string[]): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: 'subscribe', tickers }));
    }
  }

  private sendUnsubscribe(tickers: string[]): void {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ action: 'unsubscribe', tickers }));
    }
  }

  /**
   * Subscribe to price updates for given tickers.
   * Returns an observable that emits PriceUpdate for any of the requested tickers.
   */
  subscribe(tickers: string[]): Observable<PriceUpdate> {
    const upper = tickers.map(t => t.toUpperCase());
    const newTickers = upper.filter(t => !this.subscribedTickers.has(t));

    for (const t of upper) {
      this.subscribedTickers.add(t);
    }

    if (newTickers.length > 0) {
      this.sendSubscribe(newTickers);
    }

    const tickerSet = new Set(upper);
    return this.updates$.pipe(
      map(updates => updates.filter(u => tickerSet.has(u.ticker))),
      filter(updates => updates.length > 0),
      map(updates => updates[0]) // emit one at a time
      // For batch: use switchMap(updates => from(updates))
    );
  }

  /**
   * Subscribe to batch price updates (array per emission).
   */
  subscribeBatch(tickers: string[]): Observable<PriceUpdate[]> {
    const upper = tickers.map(t => t.toUpperCase());
    const newTickers = upper.filter(t => !this.subscribedTickers.has(t));

    for (const t of upper) {
      this.subscribedTickers.add(t);
    }

    if (newTickers.length > 0) {
      this.sendSubscribe(newTickers);
    }

    const tickerSet = new Set(upper);
    return this.updates$.pipe(
      map(updates => updates.filter(u => tickerSet.has(u.ticker))),
      filter(updates => updates.length > 0),
    );
  }

  /**
   * Unsubscribe from tickers no longer needed.
   */
  unsubscribe(tickers: string[]): void {
    const upper = tickers.map(t => t.toUpperCase());
    for (const t of upper) {
      this.subscribedTickers.delete(t);
    }
    this.sendUnsubscribe(upper);
  }
}
