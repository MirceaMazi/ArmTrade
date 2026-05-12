import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { ButtonModule } from 'primeng/button';
import { SkeletonModule } from 'primeng/skeleton';
import { StockService, SearchResult } from '../../services/stock.service';
import { AuthService } from '../../services/auth.service';
import { WatchlistService, WatchlistItem } from '../../services/watchlist.service';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, ButtonModule, SkeletonModule],
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
      } else {
        this.watchlist = [];
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

  removeFromWatchlist(ticker: string) {
    this.watchlistService.removeFromWatchlist(ticker).subscribe({
      next: () => {
        this.watchlist = this.watchlist.filter(w => w.ticker !== ticker);
      }
    });
  }

  search(event: any) {
    const term = event.query;
    if (term) {
      this.stockService.searchStocks(term).subscribe({
        next: (res) => {
          this.results = res.filter((item: any) => item.quoteType === 'EQUITY' || item.quoteType === 'ETF');
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
}
