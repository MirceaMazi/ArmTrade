import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { ButtonModule } from 'primeng/button';
import { SkeletonModule } from 'primeng/skeleton';
import { TagModule } from 'primeng/tag';
import { forkJoin } from 'rxjs';
import { StockService, SearchResult } from '../../services/stock.service';

interface FundamentalMetric {
  label: string;
  value1: string;
  value2: string;
  raw1: number | null;
  raw2: number | null;
  higherIsBetter: boolean;
}

@Component({
  selector: 'app-compare',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, ButtonModule, SkeletonModule, TagModule],
  templateUrl: './compare.component.html',
  styleUrl: './compare.component.css'
})
export class CompareComponent implements OnInit {
  stock1: any = null;
  stock2: any = null;
  results1: SearchResult[] = [];
  results2: SearchResult[] = [];
  loadingFundamentals = false;
  loading = false;
  comparison: any = null;
  fundamentals: FundamentalMetric[] = [];
  fund1Raw: any = null;
  fund2Raw: any = null;

  constructor(
    private stockService: StockService,
    private router: Router,
    private route: ActivatedRoute
  ) {}

  ngOnInit() {
    this.route.queryParams.subscribe(params => {
      if (params['stock1'] && params['stock2']) {
        this.stock1 = { symbol: params['stock1'], shortname: params['stock1'] };
        this.stock2 = { symbol: params['stock2'], shortname: params['stock2'] };
        this.loadFundamentals();
      }
    });
  }

  search1(event: any) {
    this.stockService.searchStocks(event.query).subscribe({
      next: (res) => this.results1 = res
    });
  }

  search2(event: any) {
    this.stockService.searchStocks(event.query).subscribe({
      next: (res) => this.results2 = res
    });
  }

  loadFundamentals() {
    if (!this.stock1?.symbol || !this.stock2?.symbol) return;
    this.loadingFundamentals = true;
    this.fundamentals = [];
    this.comparison = null;

    forkJoin({
      f1: this.stockService.getFundamentals(this.stock1.symbol),
      f2: this.stockService.getFundamentals(this.stock2.symbol)
    }).subscribe({
      next: ({ f1, f2 }) => {
        this.fund1Raw = f1?.quoteSummary?.result?.[0] || {};
        this.fund2Raw = f2?.quoteSummary?.result?.[0] || {};
        this.buildMetrics();
        this.loadingFundamentals = false;
      },
      error: () => this.loadingFundamentals = false
    });
  }

  private formatValue(v: any): string {
    if (v == null) return 'N/A';
    if (typeof v === 'object') {
      if (v.fmt != null && v.fmt !== '') return v.fmt;
      if (v.raw != null) return String(v.raw);
      return 'N/A';
    }
    return String(v) || 'N/A';
  }

  private buildMetrics() {
    const f1 = this.fund1Raw;
    const f2 = this.fund2Raw;

    const metrics: { label: string; path: string; higherIsBetter: boolean }[] = [
      { label: 'Market Cap', path: 'summaryDetail.marketCap', higherIsBetter: true },
      { label: 'Enterprise Value', path: 'defaultKeyStatistics.enterpriseValue', higherIsBetter: true },
      { label: 'Trailing P/E', path: 'summaryDetail.trailingPE', higherIsBetter: false },
      { label: 'Forward P/E', path: 'defaultKeyStatistics.forwardPE', higherIsBetter: false },
      { label: 'PEG Ratio', path: 'defaultKeyStatistics.pegRatio', higherIsBetter: false },
      { label: 'Dividend Yield', path: 'summaryDetail.dividendYield', higherIsBetter: true },
      { label: '52-Week High', path: 'summaryDetail.fiftyTwoWeekHigh', higherIsBetter: true },
      { label: '52-Week Low', path: 'summaryDetail.fiftyTwoWeekLow', higherIsBetter: false },
      { label: 'Beta', path: 'defaultKeyStatistics.beta', higherIsBetter: false },
      { label: 'Profit Margins', path: 'defaultKeyStatistics.profitMargins', higherIsBetter: true },
      { label: 'Revenue', path: 'financialData.totalRevenue', higherIsBetter: true },
      { label: 'EBITDA', path: 'financialData.ebitda', higherIsBetter: true },
      { label: 'Free Cash Flow', path: 'financialData.freeCashflow', higherIsBetter: true },
      { label: 'ROE', path: 'financialData.returnOnEquity', higherIsBetter: true },
      { label: 'Debt/Equity', path: 'financialData.debtToEquity', higherIsBetter: false },
      { label: 'Current Ratio', path: 'financialData.currentRatio', higherIsBetter: true },
      { label: 'Analyst Target', path: 'financialData.targetMeanPrice', higherIsBetter: true },
      { label: 'Recommendation', path: 'financialData.recommendationKey', higherIsBetter: true },
    ];

    this.fundamentals = metrics.map(m => {
      const v1 = this.getNestedValue(f1, m.path);
      const v2 = this.getNestedValue(f2, m.path);
      return {
        label: m.label,
        value1: this.formatValue(v1),
        value2: this.formatValue(v2),
        raw1: v1?.raw ?? null,
        raw2: v2?.raw ?? null,
        higherIsBetter: m.higherIsBetter
      };
    });
  }

  private getNestedValue(obj: any, path: string): any {
    return path.split('.').reduce((o, key) => o?.[key], obj);
  }

  getWinner(metric: FundamentalMetric): 'stock1' | 'stock2' | 'none' {
    if (metric.raw1 === null || metric.raw2 === null) return 'none';
    if (metric.raw1 === metric.raw2) return 'none';
    if (metric.higherIsBetter) {
      return metric.raw1 > metric.raw2 ? 'stock1' : 'stock2';
    } else {
      return metric.raw1 < metric.raw2 ? 'stock1' : 'stock2';
    }
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
