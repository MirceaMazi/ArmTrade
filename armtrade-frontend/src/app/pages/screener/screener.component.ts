import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { SkeletonModule } from 'primeng/skeleton';
import { StockService, ScreenerResponse } from '../../services/stock.service';

@Component({
  selector: 'app-screener',
  standalone: true,
  imports: [CommonModule, FormsModule, CardModule, ButtonModule, InputTextModule, SkeletonModule],
  templateUrl: './screener.component.html',
  styleUrl: './screener.component.css'
})
export class ScreenerComponent {
  query: string = '';
  loading: boolean = false;
  results: ScreenerResponse | null = null;

  constructor(
    private stockService: StockService,
    private router: Router
  ) {}

  search() {
    if (!this.query.trim()) return;
    this.loading = true;
    this.results = null;

    this.stockService.screenStocks(this.query).subscribe({
      next: (res) => {
        this.results = res;
        this.loading = false;
      },
      error: (err) => {
        console.error('Screener error', err);
        this.loading = false;
      }
    });
  }

  openDashboard(ticker: string) {
    this.router.navigate(['/dashboard', ticker]);
  }

  goBack() {
    this.router.navigate(['/']);
  }
}
