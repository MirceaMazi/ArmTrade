import { Component, OnInit } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { LoadingSpinnerComponent } from '../../components/loading-spinner/loading-spinner.component';
import { MarketService, IpoItem } from '../../services/market.service';

interface IpoWeekGroup {
  label: string;
  ipos: IpoItem[];
}

@Component({
  selector: 'app-ipo-calendar',
  standalone: true,
  imports: [CommonModule, FormsModule, LoadingSpinnerComponent],
  templateUrl: './ipo-calendar.component.html',
  styleUrl: './ipo-calendar.component.css'
})
export class IpoCalendarComponent implements OnInit {
  ipos: IpoItem[] = [];
  groups: IpoWeekGroup[] = [];
  loading = false;
  error = '';
  filterText = '';

  constructor(private marketService: MarketService, private router: Router) {}

  ngOnInit(): void {
    this.load();
  }

  load(): void {
    this.loading = true;
    this.error = '';
    this.marketService.getIpos().subscribe({
      next: (res) => {
        this.ipos = res;
        this.applyFilter();
        this.loading = false;
      },
      error: () => {
        this.error = 'Could not load the IPO calendar.';
        this.loading = false;
      }
    });
  }

  applyFilter(): void {
    const term = this.filterText.trim().toLowerCase();
    const filtered = term
      ? this.ipos.filter(i =>
          i.company.toLowerCase().includes(term) ||
          i.ticker.toLowerCase().includes(term))
      : this.ipos;
    this.groups = this.groupByWeek(filtered);
  }

  private groupByWeek(items: IpoItem[]): IpoWeekGroup[] {
    const map = new Map<string, IpoItem[]>();
    for (const item of items) {
      const key = this.weekLabel(item.date);
      const bucket = map.get(key);
      if (bucket) {
        bucket.push(item);
      } else {
        map.set(key, [item]);
      }
    }
    return Array.from(map.entries()).map(([label, ipos]) => ({ label, ipos }));
  }

  private weekLabel(dateStr: string): string {
    const date = new Date(dateStr);
    if (isNaN(date.getTime())) { return 'Unscheduled'; }
    const day = date.getDay();
    const monday = new Date(date);
    monday.setDate(date.getDate() - ((day + 6) % 7));
    const sunday = new Date(monday);
    sunday.setDate(monday.getDate() + 6);
    const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' };
    return `Week of ${monday.toLocaleDateString(undefined, opts)} – ${sunday.toLocaleDateString(undefined, opts)}`;
  }

  goHome(): void {
    this.router.navigate(['/']);
  }
}
