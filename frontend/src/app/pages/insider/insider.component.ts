import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { SkeletonModule } from 'primeng/skeleton';
import {
  StockService, SearchResult, InsiderRadarResponse,
  InsiderAnalysis, InsiderPattern, KeyTransaction
} from '../../services/stock.service';
import { LoadingSpinnerComponent } from '../../components/loading-spinner/loading-spinner.component';

@Component({
  selector: 'app-insider',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, SkeletonModule, LoadingSpinnerComponent],
  templateUrl: './insider.component.html',
  styleUrls: ['./insider.component.css']
})
export class InsiderComponent implements OnInit {
  selectedStock: any = null;
  searchResults: SearchResult[] = [];
  loading = false;
  hasSearched = false;
  errorMessage = '';
  currentTicker = '';

  response: InsiderRadarResponse | null = null;

  constructor(
    private stockService: StockService,
    private router: Router,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      const ticker = params['ticker'];
      if (ticker && !this.response && !this.loading) {
        this.currentTicker = ticker;
        this.selectedStock = ticker;
        this.loadInsider(ticker);
      }
    });
  }

  search(event: any) {
    const query = event.query;
    if (query && query.length >= 1) {
      this.stockService.searchStocks(query).subscribe({
        next: (results) => this.searchResults = results,
        error: () => this.searchResults = []
      });
    }
  }

  onSelect(event: any) {
    const ticker = event.value?.symbol || event.symbol;
    if (ticker) {
      this.loadInsider(ticker);
    }
  }

  onSearchKeyup(event: KeyboardEvent) {
    if (event.key === 'Enter' && this.selectedStock) {
      const ticker = typeof this.selectedStock === 'string'
        ? this.selectedStock.toUpperCase()
        : this.selectedStock.symbol;
      if (ticker) {
        this.loadInsider(ticker);
      }
    }
  }

  loadInsider(ticker: string) {
    this.loading = true;
    this.hasSearched = true;
    this.errorMessage = '';
    this.response = null;
    this.currentTicker = ticker;

    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { ticker },
      queryParamsHandling: 'merge',
      replaceUrl: true
    });

    this.stockService.getInsiderRadar(ticker).subscribe({
      next: (data) => {
        this.loading = false;
        this.response = data;
      },
      error: (err) => {
        this.loading = false;
        this.errorMessage = err.error?.error || 'Failed to fetch insider data. Try again.';
      }
    });
  }

  get analysis(): InsiderAnalysis | null {
    return this.response?.analysis || null;
  }

  get signalColor(): string {
    switch (this.analysis?.signalStrength) {
      case 'high': return '#22c55e';
      case 'moderate': return '#eab308';
      case 'low': return '#94a3b8';
      case 'neutral': return '#64748b';
      default: return '#64748b';
    }
  }

  get signalLabel(): string {
    switch (this.analysis?.signalStrength) {
      case 'high': return '🟢 High Conviction';
      case 'moderate': return '🟡 Moderate Signal';
      case 'low': return '⚪ Low Activity';
      case 'neutral': return '⚫ Neutral';
      default: return '⚫ Neutral';
    }
  }

  get signalIcon(): string {
    switch (this.analysis?.signalStrength) {
      case 'high': return '🚀';
      case 'moderate': return '👀';
      case 'low': return '😴';
      case 'neutral': return '➖';
      default: return '➖';
    }
  }

  sentimentIcon(sentiment: string): string {
    switch (sentiment) {
      case 'bullish': return '📈';
      case 'bearish': return '📉';
      default: return '➡️';
    }
  }

  sentimentColor(sentiment: string): string {
    switch (sentiment) {
      case 'bullish': return '#22c55e';
      case 'bearish': return '#ef4444';
      default: return '#94a3b8';
    }
  }

  formatValue(value: number): string {
    if (!value) return '$0';
    if (Math.abs(value) >= 1_000_000_000) return '$' + (value / 1_000_000_000).toFixed(1) + 'B';
    if (Math.abs(value) >= 1_000_000) return '$' + (value / 1_000_000).toFixed(1) + 'M';
    if (Math.abs(value) >= 1_000) return '$' + (value / 1_000).toFixed(0) + 'K';
    return '$' + value.toFixed(0);
  }

  formatShares(shares: number): string {
    if (!shares) return '0';
    if (Math.abs(shares) >= 1_000_000) return (shares / 1_000_000).toFixed(1) + 'M';
    if (Math.abs(shares) >= 1_000) return (shares / 1_000).toFixed(0) + 'K';
    return shares.toLocaleString();
  }

  transactionColor(type: string): string {
    switch (type) {
      case 'Buy': return '#22c55e';
      case 'Sell': return '#ef4444';
      default: return '#a855f7';
    }
  }

  transactionIcon(type: string): string {
    switch (type) {
      case 'Buy': return '🟢';
      case 'Sell': return '🔴';
      default: return '🟣';
    }
  }

  openSourceUrl() {
    if (this.response?.sourceUrl) {
      window.open(this.response.sourceUrl, '_blank');
    }
  }

  openDashboard(ticker: string) {
    this.router.navigate(['/dashboard', ticker]);
  }

  goHome() {
    this.router.navigate(['/']);
  }
}
