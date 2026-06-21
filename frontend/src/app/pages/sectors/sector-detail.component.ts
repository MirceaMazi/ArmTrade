import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterModule } from '@angular/router';
import { LoadingSpinnerComponent } from '../../components/loading-spinner/loading-spinner.component';
import {
  MarketService,
  SectorDetail,
  SectorSummary,
  CompanyStat
} from '../../services/market.service';

@Component({
  selector: 'app-sector-detail',
  standalone: true,
  imports: [CommonModule, RouterModule, LoadingSpinnerComponent],
  templateUrl: './sector-detail.component.html',
  styleUrl: './sector-detail.component.css'
})
export class SectorDetailComponent implements OnInit {
  slug = '';
  detail: SectorDetail | null = null;
  summary: SectorSummary | null = null;
  loadingDetail = false;
  loadingSummary = false;
  error = '';

  constructor(
    private route: ActivatedRoute,
    private router: Router,
    private marketService: MarketService
  ) {}

  ngOnInit(): void {
    this.route.paramMap.subscribe(params => {
      this.slug = params.get('sectorSlug') ?? '';
      this.loadDetail();
    });
  }

  loadDetail(): void {
    if (!this.slug) { return; }
    this.loadingDetail = true;
    this.error = '';
    this.marketService.getSectorDetails(this.slug).subscribe({
      next: (res) => {
        this.detail = res;
        this.loadingDetail = false;
        this.loadSummary(res);
      },
      error: () => {
        this.error = 'Could not load this sector.';
        this.loadingDetail = false;
      }
    });
  }

  private loadSummary(detail: SectorDetail): void {
    this.loadingSummary = true;
    const movers = detail.companies
      .slice(0, 6)
      .map(c => `${c.symbol}: ${c.dayChange >= 0 ? '+' : ''}${c.dayChange.toFixed(2)}% today, ${c.weekChange >= 0 ? '+' : ''}${c.weekChange.toFixed(2)}% this week`);
    const headlines = detail.news.slice(0, 5).map(n => n.title);

    this.marketService.getSectorSummary({ sector: detail.name, movers, headlines }).subscribe({
      next: (res) => {
        this.summary = res;
        this.loadingSummary = false;
      },
      error: () => this.loadingSummary = false
    });
  }

  openCompany(company: CompanyStat): void {
    if (company.symbol) {
      this.router.navigate(['/dashboard', company.symbol]);
    }
  }

  goHome(): void {
    this.router.navigate(['/']);
  }

  formatMarketCap(value: number): string {
    if (!value) { return '—'; }
    if (value >= 1e12) { return '$' + (value / 1e12).toFixed(2) + 'T'; }
    if (value >= 1e9) { return '$' + (value / 1e9).toFixed(2) + 'B'; }
    if (value >= 1e6) { return '$' + (value / 1e6).toFixed(2) + 'M'; }
    return '$' + value.toFixed(0);
  }

  timeAgo(published: number): string {
    if (!published) { return ''; }
    const diff = Date.now() / 1000 - published;
    if (diff < 3600) { return Math.max(1, Math.floor(diff / 60)) + 'm ago'; }
    if (diff < 86400) { return Math.floor(diff / 3600) + 'h ago'; }
    return Math.floor(diff / 86400) + 'd ago';
  }
}
