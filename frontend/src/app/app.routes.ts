import { Routes } from '@angular/router';
import { HomeComponent } from './pages/home/home.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { ScreenerComponent } from './pages/screener/screener.component';
import { LoginComponent } from './pages/login/login.component';
import { CompareComponent } from './pages/compare/compare.component';
import { EarningsComponent } from './pages/earnings/earnings.component';
import { MarketComponent } from './pages/market/market.component';

export const routes: Routes = [
  { path: '', component: HomeComponent },
  { path: 'dashboard/:ticker', component: DashboardComponent },
  { path: 'screener', component: ScreenerComponent },
  { path: 'login', component: LoginComponent },
  { path: 'compare', component: CompareComponent },
  { path: 'earnings', component: EarningsComponent },
  { path: 'market', component: MarketComponent },
  { path: '**', redirectTo: '' }
];
