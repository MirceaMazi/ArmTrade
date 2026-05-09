import { Routes } from '@angular/router';
import { HomeComponent } from './pages/home/home.component';
import { DashboardComponent } from './pages/dashboard/dashboard.component';
import { ScreenerComponent } from './pages/screener/screener.component';

export const routes: Routes = [
  { path: '', component: HomeComponent },
  { path: 'dashboard/:ticker', component: DashboardComponent },
  { path: 'screener', component: ScreenerComponent },
  { path: '**', redirectTo: '' }
];
