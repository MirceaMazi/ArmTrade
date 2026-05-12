import { Component, OnInit, ElementRef, ViewChild, OnDestroy } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { CardModule } from 'primeng/card';
import { SkeletonModule } from 'primeng/skeleton';
import { ButtonModule } from 'primeng/button';
import { Select } from 'primeng/select';
import { InputTextModule } from 'primeng/inputtext';
import { InputTextarea } from 'primeng/inputtextarea';
import { TagModule } from 'primeng/tag';
import { DialogModule } from 'primeng/dialog';
import { ProgressBarModule } from 'primeng/progressbar';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { createChart, IChartApi, CandlestickSeries } from 'lightweight-charts';
import { StockService, ArmandAnalysis, Annotation, SearchResult } from '../../services/stock.service';
import { AuthService } from '../../services/auth.service';
import { WatchlistService } from '../../services/watchlist.service';

@Component({
  selector: 'app-dashboard',
  standalone: true,
  imports: [
    CommonModule, FormsModule, CardModule, SkeletonModule, ButtonModule,
    Select, InputTextModule, InputTextarea, TagModule, DialogModule,
    AutoCompleteModule, ProgressBarModule
  ],
  templateUrl: './dashboard.component.html',
  styleUrl: './dashboard.component.css'
})
export class DashboardComponent implements OnInit, OnDestroy {
  ticker: string = '';
  loadingFundamentals: boolean = true;
  loadingAnalysis: boolean = false;
  analysisRequested: boolean = false;
  fundamentalData: any = null;
  maxFinancialValue: number = 0;
  armandAnalysis: ArmandAnalysis | null = null;
  news: any[] = [];
  loadingNews: boolean = true;
  dividends: any[] = [];
  loadingDividends: boolean = true;
  isInWatchlist: boolean = false;
  isLoggedIn: boolean = false;

  // Fundamentals dialog
  showFundamentalsDialog: boolean = false;

  // AI Advanced Controls
  showAdvanced: boolean = false;

  // Search bar in header
  searchQuery: any = null;
  searchResults: SearchResult[] = [];

  // Chart range
  selectedRange: string = '1y';
  selectedInterval: string = '1d';
  chartRanges = [
    { label: '1W', range: '5d', interval: '15m' },
    { label: '1M', range: '1mo', interval: '1d' },
    { label: '3M', range: '3mo', interval: '1d' },
    { label: '6M', range: '6mo', interval: '1d' },
    { label: '1Y', range: '1y', interval: '1d' },
    { label: '5Y', range: '5y', interval: '1wk' },
    { label: 'MAX', range: 'max', interval: '1mo' },
  ];

  // AI Persona
  selectedPersona: string = '';
  currentPersonaLabel: string = 'Balanced Analyst';
  personas = [
    { label: 'Balanced Analyst', value: '' },
    { label: 'Value Investor (Buffett)', value: 'Value Investor — Focus on intrinsic value, margin of safety, long-term compounding. Think like Warren Buffett.' },
    { label: 'Growth Investor', value: 'Growth Investor — Focus on revenue growth, market expansion, and future potential over current profitability.' },
    { label: 'Day Trader', value: 'Day Trader — Focus on short-term momentum, volume spikes, technical levels, and recent price action patterns.' },
    { label: 'Dividend Hunter', value: 'Dividend Hunter — Prioritize dividend yield, payout ratio consistency, and cash flow sustainability.' },
    { label: 'Contrarian', value: 'Contrarian Investor — Look for opportunities where the market sentiment is wrong. Go against the crowd.' }
  ];

  // What-If Scenario
  whatIfQuestion: string = '';

  // Chart
  @ViewChild('chartContainer') chartContainer!: ElementRef;
  private chart!: IChartApi;
  private candlestickSeries: any;
  private chartData: any[] = [];

  constructor(
    private route: ActivatedRoute,
    private stockService: StockService,
    private authService: AuthService,
    private watchlistService: WatchlistService,
    private router: Router,
    private location: Location
  ) {
    this.isLoggedIn = this.authService.isLoggedIn();
  }

  ngOnInit(): void {
    this.route.paramMap.subscribe(params => {
      this.ticker = params.get('ticker') || '';
      if (this.ticker) {
        this.loadDashboardData();
      } else {
        this.router.navigate(['/']);
      }
    });
  }

  ngOnDestroy(): void {
    if (this.chart) {
      this.chart.remove();
    }
  }

  goBack() {
    this.location.back();
  }

  onSearchComplete(event: any) {
    this.stockService.searchStocks(event.query).subscribe({
      next: (res) => {
        this.searchResults = res.filter((item: any) => item.quoteType === 'EQUITY' || item.quoteType === 'ETF');
      },
      error: (err) => console.error(err)
    });
  }

  onSearchSelect(event: any) {
    const item = event.value || event;
    if (item && item.symbol) {
      this.router.navigate(['/dashboard', item.symbol]);
    }
  }

  openFundamentalsDialog() {
    this.showFundamentalsDialog = true;
  }

  changeChartRange(range: string, interval: string) {
    this.selectedRange = range;
    this.selectedInterval = interval;
    if (this.chart) {
      this.chart.remove();
    }
    this.stockService.getChart(this.ticker, interval, range).subscribe({
      next: (res) => this.renderChart(res),
      error: (err) => console.error('Error fetching chart', err)
    });
  }

  runAnalysis() {
    this.loadingAnalysis = true;
    this.analysisRequested = true;
    this.armandAnalysis = null;
    
    // Find the current persona label
    const p = this.personas.find(p => p.value === this.selectedPersona);
    this.currentPersonaLabel = p ? p.label : 'Balanced Analyst';

    this.fetchAnalysis();
  }

  private loadDashboardData() {
    if (this.chart) {
      this.chart.remove();
    }
    
    this.loadingFundamentals = true;
    this.loadingNews = true;
    this.loadingDividends = true;
    this.checkWatchlist();
    // Reset AI state — user must request it
    this.loadingAnalysis = false;
    this.analysisRequested = false;
    this.armandAnalysis = null;
    this.news = [];
    this.dividends = [];

    this.stockService.getChart(this.ticker, this.selectedInterval, this.selectedRange).subscribe({
      next: (res) => this.renderChart(res),
      error: (err) => console.error('Error fetching chart', err)
    });

    this.stockService.getFundamentals(this.ticker).subscribe({
      next: (res) => {
        if (res.quoteSummary && res.quoteSummary.result && res.quoteSummary.result.length > 0) {
          this.fundamentalData = res.quoteSummary.result[0];
          
          // Calculate max value for CSS bar chart
          this.maxFinancialValue = 0;
          if (this.fundamentalData.earnings?.financialsChart?.yearly) {
            for (let year of this.fundamentalData.earnings.financialsChart.yearly) {
              const rev = year.revenue?.raw || 0;
              const earn = Math.abs(year.earnings?.raw || 0); // use absolute for scale
              if (rev > this.maxFinancialValue) this.maxFinancialValue = rev;
              if (earn > this.maxFinancialValue) this.maxFinancialValue = earn;
            }
          }
        }
        this.loadingFundamentals = false;
      },
      error: (err) => {
        console.error('Error fetching fundamentals', err);
        this.loadingFundamentals = false;
      }
    });

    this.stockService.getNews(this.ticker).subscribe({
      next: (res) => {
        this.news = res;
        this.loadingNews = false;
      },
      error: (err) => {
        console.error('Error fetching news', err);
        this.loadingNews = false;
      }
    });

    this.stockService.getDividends(this.ticker).subscribe({
      next: (res) => {
        // Format dates to locale string for UI
        this.dividends = res.map(div => ({
          ...div,
          dateString: new Date(div.date * 1000).toLocaleDateString()
        }));
        this.loadingDividends = false;
      },
      error: (err) => {
        console.error('Error fetching dividends', err);
        this.loadingDividends = false;
      }
    });
  }

  private fetchAnalysis() {
    this.stockService.getArmandAnalysis({
      ticker: this.ticker,
      persona: this.selectedPersona,
      whatIf: this.whatIfQuestion
    }).subscribe({
      next: (res) => {
        this.armandAnalysis = res;
        this.loadingAnalysis = false;
        if (res.annotations && res.annotations.length > 0) {
          this.placeAnnotations(res.annotations);
        }
      },
      error: (err) => {
        console.error('Error fetching Armand analysis', err);
        this.loadingAnalysis = false;
      }
    });
  }

  private placeAnnotations(annotations: Annotation[]) {
    if (!this.candlestickSeries || !this.chartData.length) return;

    const markers = annotations.map(ann => {
      const annDate = new Date(ann.date);
      const annTimestamp = Math.floor(annDate.getTime() / 1000);

      let closestPoint = this.chartData[0];
      let minDiff = Math.abs(this.chartData[0].time - annTimestamp);
      for (const point of this.chartData) {
        const diff = Math.abs(point.time - annTimestamp);
        if (diff < minDiff) {
          minDiff = diff;
          closestPoint = point;
        }
      }

      const isBullish = ann.type === 'bullish';
      return {
        time: closestPoint.time,
        position: isBullish ? 'belowBar' : 'aboveBar',
        color: isBullish ? '#10b981' : ann.type === 'bearish' ? '#ef4444' : '#6366f1',
        shape: isBullish ? 'arrowUp' : ann.type === 'bearish' ? 'arrowDown' : 'circle',
        text: ann.description.length > 40 ? ann.description.substring(0, 40) + '…' : ann.description,
      };
    });

    markers.sort((a: any, b: any) => a.time - b.time);
    this.candlestickSeries.setMarkers(markers);
  }

  getSentimentSeverity(sentiment: string): 'success' | 'warn' | 'danger' | 'info' {
    switch (sentiment) {
      case 'Bullish': return 'success';
      case 'Bearish': return 'danger';
      default: return 'info';
    }
  }

  private renderChart(data: any) {
    if (!data.chart || !data.chart.result || data.chart.result.length === 0) return;

    const result = data.chart.result[0];
    const timestamps = result.timestamp;
    const quote = result.indicators.quote[0];

    this.chartData = [];
    for (let i = 0; i < timestamps.length; i++) {
      if (quote.open[i] !== null && quote.high[i] !== null && quote.low[i] !== null && quote.close[i] !== null) {
        this.chartData.push({
          time: timestamps[i],
          open: quote.open[i],
          high: quote.high[i],
          low: quote.low[i],
          close: quote.close[i],
        });
      }
    }

    if (!this.chartContainer) return;

    this.chart = createChart(this.chartContainer.nativeElement, {
      autoSize: true,
      layout: {
        background: { color: 'transparent' },
        textColor: '#64748b',
        fontFamily: 'Inter',
      },
      grid: {
        vertLines: { color: 'rgba(99, 102, 241, 0.06)' },
        horzLines: { color: 'rgba(99, 102, 241, 0.06)' },
      },
      crosshair: {
        mode: 1,
        vertLine: { color: 'rgba(99, 102, 241, 0.3)', labelBackgroundColor: '#6366f1' },
        horzLine: { color: 'rgba(99, 102, 241, 0.3)', labelBackgroundColor: '#6366f1' },
      },
      timeScale: {
        borderColor: 'rgba(99, 102, 241, 0.1)',
        fixLeftEdge: true,
        fixRightEdge: true,
      },
      rightPriceScale: {
        borderColor: 'rgba(99, 102, 241, 0.1)',
      },
    });

    this.candlestickSeries = this.chart.addSeries(CandlestickSeries, {
      upColor: '#10b981',
      downColor: '#ef4444',
      borderVisible: false,
      wickUpColor: '#10b981',
      wickDownColor: '#ef4444',
    });

    this.candlestickSeries.setData(this.chartData);
    this.chart.timeScale().fitContent();
  }

  abs(val: number): number {
    return Math.abs(val);
  }

  toggleWatchlist() {
    if (!this.isLoggedIn) {
      this.router.navigate(['/login']);
      return;
    }
    if (this.isInWatchlist) {
      this.watchlistService.removeFromWatchlist(this.ticker).subscribe({
        next: () => this.isInWatchlist = false
      });
    } else {
      this.watchlistService.addToWatchlist(this.ticker).subscribe({
        next: () => this.isInWatchlist = true
      });
    }
  }

  private checkWatchlist() {
    if (!this.isLoggedIn) return;
    this.watchlistService.getWatchlist().subscribe({
      next: (items) => {
        this.isInWatchlist = items.some(i => i.ticker === this.ticker);
      }
    });
  }
}
