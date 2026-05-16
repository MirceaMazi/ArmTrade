import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { SkeletonModule } from 'primeng/skeleton';
import { CheckboxModule } from 'primeng/checkbox';
import { StockService, ScreenerResponse } from '../../services/stock.service';

@Component({
  selector: 'app-screener',
  standalone: true,
  imports: [CommonModule, FormsModule, CardModule, ButtonModule, InputTextModule, SkeletonModule, CheckboxModule],
  templateUrl: './screener.component.html',
  styleUrl: './screener.component.css'
})
export class ScreenerComponent implements OnInit {
  query: string = '';
  loading: boolean = false;
  results: ScreenerResponse | null = null;
  selectedTickers: Set<string> = new Set();

  constructor(
    private stockService: StockService,
    private router: Router
  ) {}

  ngOnInit() {
    const saved = sessionStorage.getItem('screener_state');
    if (saved) {
      try {
        const state = JSON.parse(saved);
        this.query = state.query || '';
        this.results = state.results || null;
      } catch {}
    }
  }

  private saveState() {
    sessionStorage.setItem('screener_state', JSON.stringify({
      query: this.query,
      results: this.results
    }));
  }

  search() {
    if (!this.query.trim()) return;
    this.loading = true;
    this.results = null;
    this.selectedTickers.clear();

    this.stockService.screenStocks(this.query).subscribe({
      next: (res) => {
        this.results = res;
        this.loading = false;
        this.saveState();
      },
      error: (err) => {
        console.error('Screener error', err);
        this.loading = false;
      }
    });
  }

  toggleSelection(ticker: string) {
    if (this.selectedTickers.has(ticker)) {
      this.selectedTickers.delete(ticker);
    } else if (this.selectedTickers.size < 2) {
      this.selectedTickers.add(ticker);
    }
  }

  isSelected(ticker: string): boolean {
    return this.selectedTickers.has(ticker);
  }

  canCompare(): boolean {
    return this.selectedTickers.size === 2;
  }

  compareSelected() {
    const tickers = Array.from(this.selectedTickers);
    this.router.navigate(['/compare'], { queryParams: { stock1: tickers[0], stock2: tickers[1] } });
  }

  openDashboard(ticker: string) {
    this.router.navigate(['/dashboard', ticker]);
  }

  goBack() {
    this.router.navigate(['/']);
  }
}
