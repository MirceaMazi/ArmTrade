import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { ButtonModule } from 'primeng/button';
import { StockService, SearchResult } from '../../services/stock.service';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, ButtonModule],
  templateUrl: './home.component.html',
  styleUrl: './home.component.css'
})
export class HomeComponent implements OnInit {
  query: string = '';
  results: SearchResult[] = [];
  selectedStock: any;

  constructor(private stockService: StockService, private router: Router) {}

  ngOnInit(): void {}

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

  openScreener() {
    this.router.navigate(['/screener']);
  }
}
