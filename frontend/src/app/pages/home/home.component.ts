import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { ButtonModule } from 'primeng/button';
import { SkeletonModule } from 'primeng/skeleton';
import { DialogModule } from 'primeng/dialog';
import { InputNumberModule } from 'primeng/inputnumber';
import { StockService, SearchResult } from '../../services/stock.service';
import { AuthService } from '../../services/auth.service';
import { WatchlistService, WatchlistItem } from '../../services/watchlist.service';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, ButtonModule, SkeletonModule, DialogModule, InputNumberModule],
  templateUrl: './home.component.html',
  styleUrl: './home.component.css'
})
export class HomeComponent implements OnInit {
  query: string = '';
  results: SearchResult[] = [];
  selectedStock: any;

  isLoggedIn = false;
  username = '';
  watchlist: WatchlistItem[] = [];
  loadingWatchlist = false;
  savedAnalyses: any[] = [];
  loadingSavedAnalyses = false;

  // Portfolio dialog
  portfolioDialogVisible = false;
  editingItemId = '';
  editingTicker = '';
  editBuyPrice: number | null = null;
  editQuantity: number | null = null;
  editBuyDate: string = '';

  // Analysis detail dialog
  analysisDialogVisible = false;
  selectedAnalysis: any = null;

  constructor(
    private stockService: StockService,
    private authService: AuthService,
    private watchlistService: WatchlistService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.authService.isLoggedIn$.subscribe(val => {
      this.isLoggedIn = val;
      if (val) {
        this.loadWatchlist();
        this.loadSavedAnalyses();
      } else {
        this.watchlist = [];
        this.savedAnalyses = [];
      }
    });
    this.authService.username$.subscribe(val => this.username = val);
  }

  loadWatchlist() {
    this.loadingWatchlist = true;
    this.watchlistService.getWatchlist().subscribe({
      next: (res) => {
        this.watchlist = res;
        this.loadingWatchlist = false;
      },
      error: () => this.loadingWatchlist = false
    });
  }

  removeFromWatchlist(id: string) {
    this.watchlistService.removeFromWatchlist(id).subscribe({
      next: () => {
        this.watchlist = this.watchlist.filter(w => w.id !== id);
      }
    });
  }

  search(event: any) {
    const term = event.query;
    if (term) {
      this.stockService.searchStocks(term).subscribe({
        next: (res) => {
          this.results = res;
        },
        error: (err) => console.error(err)
      });
    }
  }

  onSelect(event: any) {
    // PrimeNG wraps the selected item in the 'value' property
    const selectedItem = event.value || event;
    if (selectedItem && selectedItem.symbol) {
      this.router.navigate(['/dashboard', selectedItem.symbol]);
    }
  }

  openScreener() { this.router.navigate(['/screener']); }
  openCompare() { this.router.navigate(['/compare']); }
  openEarnings() { this.router.navigate(['/earnings']); }
  openMarket() { this.router.navigate(['/market']); }
  openLogin() { this.router.navigate(['/login']); }
  logout() { this.authService.logout(); }
  openDashboard(ticker: string) { this.router.navigate(['/dashboard', ticker]); }

  loadSavedAnalyses() {
    this.loadingSavedAnalyses = true;
    this.stockService.getSavedAnalyses().subscribe({
      next: (res) => {
        this.savedAnalyses = res;
        this.loadingSavedAnalyses = false;
      },
      error: () => this.loadingSavedAnalyses = false
    });
  }

  openPortfolioDialog(item: WatchlistItem) {
    this.editingItemId = item.id;
    this.editingTicker = item.ticker;
    this.editBuyPrice = item.buyPrice ?? null;
    this.editQuantity = item.quantity ?? null;
    this.editBuyDate = item.buyDate ?? '';
    this.portfolioDialogVisible = true;
  }

  savePortfolio() {
    const data: any = {};
    if (this.editBuyPrice != null) data.buyPrice = this.editBuyPrice;
    if (this.editQuantity != null) data.quantity = this.editQuantity;
    if (this.editBuyDate) data.buyDate = this.editBuyDate;

    this.watchlistService.updatePortfolio(this.editingItemId, data).subscribe({
      next: () => {
        this.portfolioDialogVisible = false;
        this.loadWatchlist();
      }
    });
  }

  openAnalysisDialog(analysis: any) {
    this.selectedAnalysis = analysis;
    this.analysisDialogVisible = true;
  }

  getPnL(item: WatchlistItem): number | null {
    if (item.buyPrice && item.quantity) {
      return (item.price - item.buyPrice) * item.quantity;
    }
    return null;
  }

  getPnLPercent(item: WatchlistItem): number | null {
    if (item.buyPrice && item.buyPrice > 0) {
      return ((item.price - item.buyPrice) / item.buyPrice) * 100;
    }
    return null;
  }

  getTotalPortfolioValue(): number {
    return this.watchlist.reduce((sum, item) => {
      if (item.quantity) {
        return sum + item.price * item.quantity;
      }
      return sum;
    }, 0);
  }

  getTotalPnL(): number {
    return this.watchlist.reduce((sum, item) => {
      const pnl = this.getPnL(item);
      return sum + (pnl ?? 0);
    }, 0);
  }

  hasPortfolioData(): boolean {
    return this.watchlist.some(item => item.buyPrice != null && item.quantity != null);
  }
}
