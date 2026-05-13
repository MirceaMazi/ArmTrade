import { Component, OnInit } from '@angular/core';
import { CommonModule, DecimalPipe } from '@angular/common';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { SkeletonModule } from 'primeng/skeleton';
import { environment } from '../../../environments/environment';

@Component({
  selector: 'app-market',
  standalone: true,
  imports: [CommonModule, SkeletonModule, DecimalPipe],
  templateUrl: './market.component.html',
  styleUrl: './market.component.css'
})
export class MarketComponent implements OnInit {
  private apiUrl = `${environment.apiUrl}/market`;

  sectors: any[] = [];
  movers: any = null;
  macros: any[] = [];
  loadingSectors = true;
  loadingMovers = true;
  loadingMacro = true;

  constructor(private http: HttpClient, private router: Router) {}

  ngOnInit() {
    this.http.get<any[]>(`${this.apiUrl}/sectors`).subscribe({
      next: (res) => { this.sectors = res; this.loadingSectors = false; },
      error: () => this.loadingSectors = false
    });

    this.http.get<any>(`${this.apiUrl}/movers`).subscribe({
      next: (res) => { this.movers = res; this.loadingMovers = false; },
      error: () => this.loadingMovers = false
    });

    this.http.get<any[]>(`${this.apiUrl}/macro`).subscribe({
      next: (res) => { this.macros = res; this.loadingMacro = false; },
      error: () => this.loadingMacro = false
    });
  }

  getSectorColor(change: number): string {
    if (change > 2) return '#22c55e';
    if (change > 0.5) return '#4ade80';
    if (change > 0) return '#86efac';
    if (change > -0.5) return '#fca5a5';
    if (change > -2) return '#f87171';
    return '#ef4444';
  }

  getSectorBg(change: number): string {
    if (change > 0) return `rgba(34, 197, 94, ${Math.min(Math.abs(change) * 0.04, 0.2)})`;
    return `rgba(239, 68, 68, ${Math.min(Math.abs(change) * 0.04, 0.2)})`;
  }

  openDashboard(ticker: string) {
    this.router.navigate(['/dashboard', ticker]);
  }

  goBack() {
    this.router.navigate(['/']);
  }
}
