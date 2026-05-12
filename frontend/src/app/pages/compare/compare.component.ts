import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { ButtonModule } from 'primeng/button';
import { SkeletonModule } from 'primeng/skeleton';
import { TagModule } from 'primeng/tag';
import { StockService, SearchResult } from '../../services/stock.service';

@Component({
  selector: 'app-compare',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, ButtonModule, SkeletonModule, TagModule],
  templateUrl: './compare.component.html',
  styleUrl: './compare.component.css'
})
export class CompareComponent {
  stock1: any = null;
  stock2: any = null;
  results1: SearchResult[] = [];
  results2: SearchResult[] = [];
  loading = false;
  comparison: any = null;

  constructor(private stockService: StockService, private router: Router) {}

  search1(event: any) {
    this.stockService.searchStocks(event.query).subscribe({
      next: (res) => this.results1 = res.filter((i: any) => i.quoteType === 'EQUITY' || i.quoteType === 'ETF')
    });
  }

  search2(event: any) {
    this.stockService.searchStocks(event.query).subscribe({
      next: (res) => this.results2 = res.filter((i: any) => i.quoteType === 'EQUITY' || i.quoteType === 'ETF')
    });
  }

  compare() {
    if (!this.stock1?.symbol || !this.stock2?.symbol) return;
    this.loading = true;
    this.comparison = null;

    this.stockService.compareStocks(this.stock1.symbol, this.stock2.symbol).subscribe({
      next: (res) => {
        this.comparison = res;
        this.loading = false;
      },
      error: () => this.loading = false
    });
  }

  goBack() {
    this.router.navigate(['/']);
  }
}
