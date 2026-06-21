import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { LoadingSpinnerComponent } from '../../components/loading-spinner/loading-spinner.component';
import { MarketService, EarningsItem } from '../../services/market.service';

interface EarningsGroup {
  company: string;
  date: string;
  time: string;
  epsEstimate: number;
  hasEps: boolean;
  epsForward: boolean;
  variants: EarningsItem[];
}

@Component({
  selector: 'app-earnings-calendar',
  standalone: true,
  imports: [CommonModule, FormsModule, LoadingSpinnerComponent],
  templateUrl: './earnings-calendar.component.html',
  styleUrl: './earnings-calendar.component.css'
})
export class EarningsCalendarComponent implements OnInit {
  earnings: EarningsItem[] = [];
  groups: EarningsGroup[] = [];
  expandedCompanies = new Set<string>();
  loading = false;
  error = '';
  searchText = '';
  fromDate = '';
  toDate = '';

  constructor(private marketService: MarketService, private router: Router) {}

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading = true;
    this.error = '';
    this.marketService.getEarningsCalendar().subscribe({
      next: (res) => {
        this.earnings = res;
        this.applyFilter();
        this.loading = false;
      },
      error: () => {
        this.error = 'Could not load the earnings calendar.';
        this.loading = false;
      }
    });
  }

  applyFilter(): void {
    const term = this.searchText.trim().toLowerCase();
    const filtered = this.earnings.filter(e => {
      const matchesText = !term ||
        e.company.toLowerCase().includes(term) ||
        e.ticker.toLowerCase().includes(term);
      const matchesFrom = !this.fromDate || e.date >= this.fromDate;
      const matchesTo = !this.toDate || e.date <= this.toDate;
      return matchesText && matchesFrom && matchesTo;
    });
    this.groups = this.groupByCompany(filtered);
  }

  private groupByCompany(items: EarningsItem[]): EarningsGroup[] {
    const map = new Map<string, EarningsGroup>();
    for (const item of items) {
      const key = item.company || item.ticker;
      const existing = map.get(key);
      if (existing) {
        existing.variants.push(item);
      } else {
        map.set(key, {
          company: item.company,
          date: item.date,
          time: item.time,
          epsEstimate: item.epsEstimate,
          hasEps: item.hasEps,
          epsForward: item.epsForward,
          variants: [item]
        });
      }
    }
    return Array.from(map.values());
  }

  onGroupClick(group: EarningsGroup): void {
    if (group.variants.length === 1) {
      this.openCompany(group.variants[0]);
      return;
    }
    if (this.expandedCompanies.has(group.company)) {
      this.expandedCompanies.delete(group.company);
    } else {
      this.expandedCompanies.add(group.company);
    }
  }

  isExpanded(group: EarningsGroup): boolean {
    return this.expandedCompanies.has(group.company);
  }

  timeLabel(time: string): string {
    switch (time) {
      case 'pre-market': return 'Pre-market';
      case 'after-hours': return 'After-hours';
      case 'during-market': return 'During market';
      default: return 'TBD';
    }
  }

  openCompany(item: EarningsItem): void {
    if (item.ticker) {
      this.router.navigate(['/dashboard', item.ticker]);
    }
  }

  goHome(): void {
    this.router.navigate(['/']);
  }
}
