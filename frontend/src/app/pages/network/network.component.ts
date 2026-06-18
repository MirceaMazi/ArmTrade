import {
  Component, OnInit, OnDestroy, ViewChild, ElementRef,
  AfterViewInit, HostListener, NgZone
} from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { AutoCompleteModule } from 'primeng/autocomplete';
import { SkeletonModule } from 'primeng/skeleton';
import { StockService, SearchResult, NetworkResponse, NetworkNode, NetworkEdge } from '../../services/stock.service';

interface GraphNode {
  ticker: string;
  name: string;
  sector: string;
  description: string;
  x: number;
  y: number;
  vx: number;
  vy: number;
  radius: number;
  isCenter: boolean;
  isRipple: boolean;   // indirect / ripple-effect node
  opacity: number;
  targetOpacity: number;
  appearDelay: number;
}

interface GraphEdge {
  source: GraphNode;
  target: GraphNode;
  relationship: string;
  label: string;
  opacity: number;
  targetOpacity: number;
  drawProgress: number;
}

const RELATIONSHIP_COLORS: Record<string, string> = {
  supplier: '#3b82f6',
  competitor: '#f97316',
  customer: '#22c55e',
  partner: '#a855f7',
  subsidiary: '#eab308',
  ripple: '#06b6d4'
};

const RELATIONSHIP_ICONS: Record<string, string> = {
  supplier: '📦',
  competitor: '⚔️',
  customer: '🛒',
  partner: '🤝',
  subsidiary: '🏢',
  ripple: '🌊'
};

@Component({
  selector: 'app-network',
  standalone: true,
  imports: [CommonModule, FormsModule, AutoCompleteModule, SkeletonModule],
  templateUrl: './network.component.html',
  styleUrls: ['./network.component.css']
})
export class NetworkComponent implements OnInit, AfterViewInit, OnDestroy {
  @ViewChild('graphCanvas', { static: false }) canvasRef!: ElementRef<HTMLCanvasElement>;

  selectedStock: any = null;
  searchResults: SearchResult[] = [];
  loading = false;
  hasSearched = false;
  errorMessage = '';
  networkData: NetworkResponse | null = null;
  currentTicker = '';    // persisted ticker for back-navigation

  // Graph state
  private nodes: GraphNode[] = [];
  private edges: GraphEdge[] = [];
  private canvas!: HTMLCanvasElement;
  private ctx!: CanvasRenderingContext2D;
  private animationId: number = 0;
  private simulationRunning = false;
  private simulationAlpha = 1;
  private startTime = 0;

  // Pan / Camera state (middle-mouse drag like Figma)
  private panX = 0;
  private panY = 0;
  private zoom = 1;
  private isPanning = false;
  private panStartX = 0;
  private panStartY = 0;
  private panOriginX = 0;
  private panOriginY = 0;

  // Interaction state
  private hoveredNode: GraphNode | null = null;
  private hoveredEdge: GraphEdge | null = null;
  private draggedNode: GraphNode | null = null;
  private isDragging = false;
  private dragStartX = 0;
  private dragStartY = 0;
  private dragMoved = false;   // track if mouse actually moved during drag
  private mouseX = 0;
  private mouseY = 0;
  private devicePixelRatio = 1;

  // Tooltip
  tooltipVisible = false;
  tooltipX = 0;
  tooltipY = 0;
  tooltipTicker = '';
  tooltipName = '';
  tooltipDescription = '';
  tooltipSector = '';
  tooltipRelationship = '';
  tooltipRelLabel = '';

  constructor(
    private stockService: StockService,
    private router: Router,
    private route: ActivatedRoute,
    private ngZone: NgZone
  ) {}

  ngOnInit() {
    // Restore from query param on back-navigation
    this.route.queryParams.subscribe(params => {
      const ticker = params['ticker'];
      if (ticker) {
        // If we have a cache for this exact ticker, restore it!
        const cache = this.stockService.networkCache;
        if (cache && cache.ticker === ticker) {
          this.currentTicker = ticker;
          this.selectedStock = ticker;
          this.hasSearched = true;
          this.networkData = cache.data;
          this.nodes = cache.nodes;
          this.edges = cache.edges;
          this.panX = cache.panX;
          this.panY = cache.panY;
          this.zoom = cache.zoom;

          // Allow DOM to render the canvas first
          setTimeout(() => {
            if (this.canvasRef) {
              this.canvas = this.canvasRef.nativeElement;
              this.ctx = this.canvas.getContext('2d')!;
              this.devicePixelRatio = window.devicePixelRatio || 1;
              this.resizeCanvas();
              this.simulationAlpha = 0; // ensure physics stops so positions stay exact
              this.simulationRunning = false;
              this.render();
            }
          }, 50);
          return;
        }

        if (!this.networkData && !this.loading) {
          this.currentTicker = ticker;
          this.selectedStock = ticker;
          this.loadNetwork(ticker);
        }
      }
    });
  }

  ngAfterViewInit() {
    // Canvas will be set up after data loads
  }

  ngOnDestroy() {
    if (this.animationId) {
      cancelAnimationFrame(this.animationId);
    }
    // Save cache before component unmounts
    if (this.networkData && this.currentTicker) {
      this.stockService.networkCache = {
        ticker: this.currentTicker,
        data: this.networkData,
        nodes: this.nodes,
        edges: this.edges,
        panX: this.panX,
        panY: this.panY,
        zoom: this.zoom
      };
    }
  }

  search(event: any) {
    const query = event.query;
    if (query && query.length >= 1) {
      this.stockService.searchStocks(query).subscribe({
        next: (results) => this.searchResults = results,
        error: () => this.searchResults = []
      });
    }
  }

  onSelect(event: any) {
    const ticker = event.value?.symbol || event.symbol;
    if (ticker) {
      this.loadNetwork(ticker);
    }
  }

  onSearchKeyup(event: KeyboardEvent) {
    if (event.key === 'Enter' && this.selectedStock) {
      const ticker = typeof this.selectedStock === 'string'
        ? this.selectedStock.toUpperCase()
        : this.selectedStock.symbol;
      if (ticker) {
        this.loadNetwork(ticker);
      }
    }
  }

  loadNetwork(ticker: string) {
    this.loading = true;
    this.hasSearched = true;
    this.errorMessage = '';
    this.networkData = null;
    this.tooltipVisible = false;
    this.currentTicker = ticker;

    // Persist ticker in URL so back-navigation restores it
    this.router.navigate([], {
      relativeTo: this.route,
      queryParams: { ticker },
      queryParamsHandling: 'merge',
      replaceUrl: true
    });

    this.stockService.getNetwork(ticker).subscribe({
      next: (data) => {
        this.loading = false;
        this.networkData = data;
        // Allow DOM to render the canvas first
        setTimeout(() => this.initGraph(data), 50);
      },
      error: (err) => {
        this.loading = false;
        this.errorMessage = err.error?.error || 'Failed to discover network. Try again.';
      }
    });
  }

  private initGraph(data: NetworkResponse) {
    if (!this.canvasRef) return;
    this.canvas = this.canvasRef.nativeElement;
    this.ctx = this.canvas.getContext('2d')!;
    this.devicePixelRatio = window.devicePixelRatio || 1;
    this.resizeCanvas();

    // Reset pan and zoom
    this.panX = 0;
    this.panY = 0;
    this.zoom = 1;

    const w = this.canvas.width / this.devicePixelRatio;
    const h = this.canvas.height / this.devicePixelRatio;
    const centerX = w / 2;
    const centerY = h / 2;

    // Determine which tickers are "ripple" nodes (connected only via ripple edges)
    const rippleTickers = new Set<string>();
    for (const e of data.edges) {
      if (e.relationship === 'ripple') {
        rippleTickers.add(e.from);
        rippleTickers.add(e.to);
      }
    }
    // A node is only a ripple node if ALL its edges are ripple-type
    const nonRippleTickers = new Set<string>();
    for (const e of data.edges) {
      if (e.relationship !== 'ripple') {
        nonRippleTickers.add(e.from);
        nonRippleTickers.add(e.to);
      }
    }

    // Build nodes
    this.nodes = data.nodes.map((n, i) => {
      const isCenter = n.ticker === data.centerTicker;
      const isRipple = rippleTickers.has(n.ticker) && !nonRippleTickers.has(n.ticker) && !isCenter;
      const angle = (2 * Math.PI * i) / data.nodes.length;
      const spread = Math.min(w, h) * (isRipple ? 0.38 : 0.28);
      return {
        ticker: n.ticker,
        name: n.name,
        sector: n.sector,
        description: n.description,
        x: isCenter ? centerX : centerX + Math.cos(angle) * spread + (Math.random() - 0.5) * 40,
        y: isCenter ? centerY : centerY + Math.sin(angle) * spread + (Math.random() - 0.5) * 40,
        vx: 0,
        vy: 0,
        radius: isCenter ? 40 : (isRipple ? 25 : 30),
        isCenter,
        isRipple,
        opacity: 0,
        targetOpacity: 1,
        appearDelay: isCenter ? 0 : (isRipple ? 500 + i * 80 : 200 + i * 80)
      };
    });

    // Build edges
    this.edges = data.edges.map((e, i) => {
      const source = this.nodes.find(n => n.ticker === e.from);
      const target = this.nodes.find(n => n.ticker === e.to);
      if (!source || !target) return null!;
      return {
        source,
        target,
        relationship: e.relationship,
        label: e.label,
        opacity: 0,
        targetOpacity: 0.7,
        drawProgress: 0
      };
    }).filter(e => e !== null);

    this.startTime = performance.now();
    this.simulationAlpha = 1;
    this.simulationRunning = true;

    // Run animation loop outside Angular zone for performance
    if (this.animationId) cancelAnimationFrame(this.animationId);
    this.ngZone.runOutsideAngular(() => {
      this.animate();
    });
  }

  private resizeCanvas() {
    const container = this.canvas.parentElement!;
    const rect = container.getBoundingClientRect();
    this.canvas.width = rect.width * this.devicePixelRatio;
    this.canvas.height = rect.height * this.devicePixelRatio;
    this.canvas.style.width = rect.width + 'px';
    this.canvas.style.height = rect.height + 'px';
    this.ctx.setTransform(this.devicePixelRatio, 0, 0, this.devicePixelRatio, 0, 0);
  }

  @HostListener('window:resize')
  onResize() {
    if (this.canvas && this.ctx) {
      this.resizeCanvas();
    }
  }

  private animate() {
    const now = performance.now();
    const elapsed = now - this.startTime;

    this.updatePhysics();
    this.updateAnimations(elapsed);
    this.render();

    if (this.simulationRunning || this.nodes.some(n => n.opacity < n.targetOpacity)) {
      this.animationId = requestAnimationFrame(() => this.animate());
    } else {
      this.animationId = 0;
    }
  }

  private ensureAnimating() {
    if (!this.animationId) {
      this.ngZone.runOutsideAngular(() => this.animate());
    }
  }

  private updatePhysics() {
    if (this.simulationAlpha < 0.001) {
      this.simulationRunning = false;
      return;
    }

    const w = this.canvas.width / this.devicePixelRatio;
    const h = this.canvas.height / this.devicePixelRatio;
    const centerX = w / 2;
    const centerY = h / 2;

    // Repulsion between nodes
    for (let i = 0; i < this.nodes.length; i++) {
      for (let j = i + 1; j < this.nodes.length; j++) {
        const a = this.nodes[i];
        const b = this.nodes[j];
        let dx = b.x - a.x;
        let dy = b.y - a.y;
        let dist = Math.sqrt(dx * dx + dy * dy) || 1;
        const minDist = 120;
        const force = (minDist * minDist) / (dist * dist) * 2;
        const fx = (dx / dist) * force * this.simulationAlpha;
        const fy = (dy / dist) * force * this.simulationAlpha;
        a.vx -= fx;
        a.vy -= fy;
        b.vx += fx;
        b.vy += fy;
      }
    }

    // Attraction along edges (spring force)
    for (const edge of this.edges) {
      const a = edge.source;
      const b = edge.target;
      let dx = b.x - a.x;
      let dy = b.y - a.y;
      let dist = Math.sqrt(dx * dx + dy * dy) || 1;
      const targetDist = edge.relationship === 'ripple' ? 220 : 180;
      const force = (dist - targetDist) * 0.005 * this.simulationAlpha;
      const fx = (dx / dist) * force;
      const fy = (dy / dist) * force;
      a.vx += fx;
      a.vy += fy;
      b.vx -= fx;
      b.vy -= fy;
    }

    // Center gravity
    for (const node of this.nodes) {
      const dx = centerX - node.x;
      const dy = centerY - node.y;
      node.vx += dx * 0.001 * this.simulationAlpha;
      node.vy += dy * 0.001 * this.simulationAlpha;
    }

    // Apply velocity and damping
    for (const node of this.nodes) {
      if (this.draggedNode === node) continue;
      node.vx *= 0.85;
      node.vy *= 0.85;
      node.x += node.vx;
      node.y += node.vy;
    }

    this.simulationAlpha *= 0.995;
  }

  private updateAnimations(elapsed: number) {
    for (const node of this.nodes) {
      if (elapsed > node.appearDelay) {
        node.opacity = Math.min(node.opacity + 0.04, node.targetOpacity);
      }
    }

    const edgeDelay = 600;
    for (let i = 0; i < this.edges.length; i++) {
      const e = this.edges[i];
      if (elapsed > edgeDelay + i * 50) {
        e.opacity = Math.min(e.opacity + 0.03, e.targetOpacity);
        e.drawProgress = Math.min(e.drawProgress + 0.03, 1);
      }
    }
  }

  private render() {
    const w = this.canvas.width / this.devicePixelRatio;
    const h = this.canvas.height / this.devicePixelRatio;

    this.ctx.save();
    this.ctx.clearRect(-5000, -5000, 10000, 10000);

    // Apply pan and zoom transform
    this.ctx.translate(this.panX, this.panY);
    this.ctx.scale(this.zoom, this.zoom);

    // Draw edges
    for (const edge of this.edges) {
      if (edge.opacity <= 0) continue;
      this.drawEdge(edge);
    }

    // Draw nodes (ripple nodes first so direct nodes draw on top)
    for (const node of this.nodes) {
      if (node.opacity <= 0) continue;
      if (node.isRipple) this.drawNode(node);
    }
    for (const node of this.nodes) {
      if (node.opacity <= 0) continue;
      if (!node.isRipple) this.drawNode(node);
    }

    this.ctx.restore();
  }

  private drawEdge(edge: GraphEdge) {
    const ctx = this.ctx;
    const color = RELATIONSHIP_COLORS[edge.relationship] || '#64748b';
    const isHovered = edge === this.hoveredEdge;
    const isRipple = edge.relationship === 'ripple';

    const sx = edge.source.x;
    const sy = edge.source.y;
    const tx = edge.target.x;
    const ty = edge.target.y;

    // Calculate a curved path using a quadratic bezier
    const mx = (sx + tx) / 2;
    const my = (sy + ty) / 2;
    const dx = tx - sx;
    const dy = ty - sy;
    const len = Math.sqrt(dx * dx + dy * dy) || 1;
    const offset = len * 0.15;
    const nx = -dy / len;
    const ny = dx / len;
    const cpx = mx + nx * offset;
    const cpy = my + ny * offset;

    // Draw progress animation
    ctx.save();
    ctx.globalAlpha = edge.opacity * (isHovered ? 1 : 0.6);
    ctx.strokeStyle = color;
    ctx.lineWidth = isHovered ? 3 : 1.5;

    // Ripple edges are dashed
    if (isRipple && !isHovered) {
      ctx.setLineDash([6, 4]);
    }

    ctx.beginPath();
    ctx.moveTo(sx, sy);

    if (edge.drawProgress < 1) {
      const t = edge.drawProgress;
      const ix = (1 - t) * (1 - t) * sx + 2 * (1 - t) * t * cpx + t * t * tx;
      const iy = (1 - t) * (1 - t) * sy + 2 * (1 - t) * t * cpy + t * t * ty;
      ctx.quadraticCurveTo(
        sx + (cpx - sx) * t,
        sy + (cpy - sy) * t,
        ix, iy
      );
    } else {
      ctx.quadraticCurveTo(cpx, cpy, tx, ty);
    }

    ctx.stroke();

    // Draw label on hover
    if (isHovered && edge.drawProgress >= 1) {
      const labelT = 0.5;
      const lx = (1 - labelT) * (1 - labelT) * sx + 2 * (1 - labelT) * labelT * cpx + labelT * labelT * tx;
      const ly = (1 - labelT) * (1 - labelT) * sy + 2 * (1 - labelT) * labelT * cpy + labelT * labelT * ty;

      ctx.globalAlpha = 1;
      ctx.setLineDash([]);
      ctx.font = '600 11px Inter, sans-serif';
      const text = edge.label;
      const tm = ctx.measureText(text);
      const pw = 8;

      // Background pill
      ctx.fillStyle = 'rgba(15, 23, 42, 0.95)';
      const pillW = tm.width + pw * 2;
      const pillH = 20;
      const pillX = lx - pillW / 2;
      const pillY = ly - pillH / 2;
      ctx.beginPath();
      ctx.roundRect(pillX, pillY, pillW, pillH, 6);
      ctx.fill();
      ctx.strokeStyle = color;
      ctx.lineWidth = 1;
      ctx.stroke();

      // Text
      ctx.fillStyle = color;
      ctx.textAlign = 'center';
      ctx.textBaseline = 'middle';
      ctx.fillText(text, lx, ly);
    }

    ctx.restore();
  }

  private drawNode(node: GraphNode) {
    const ctx = this.ctx;
    const isHovered = node === this.hoveredNode;
    const r = node.radius + (isHovered ? 4 : 0);

    ctx.save();
    ctx.globalAlpha = node.opacity;

    // Glow for center node
    if (node.isCenter) {
      const pulsePhase = Math.sin(performance.now() / 800) * 0.3 + 0.7;
      const glowGrad = ctx.createRadialGradient(node.x, node.y, r * 0.5, node.x, node.y, r * 2.5);
      glowGrad.addColorStop(0, `rgba(99, 102, 241, ${0.3 * pulsePhase})`);
      glowGrad.addColorStop(1, 'rgba(99, 102, 241, 0)');
      ctx.fillStyle = glowGrad;
      ctx.beginPath();
      ctx.arc(node.x, node.y, r * 2.5, 0, Math.PI * 2);
      ctx.fill();
    }

    // Node background
    const bgGrad = ctx.createRadialGradient(node.x - r * 0.3, node.y - r * 0.3, 0, node.x, node.y, r);
    if (node.isCenter) {
      bgGrad.addColorStop(0, 'rgba(99, 102, 241, 0.35)');
      bgGrad.addColorStop(1, 'rgba(49, 46, 129, 0.8)');
    } else if (node.isRipple) {
      bgGrad.addColorStop(0, 'rgba(6, 182, 212, 0.15)');
      bgGrad.addColorStop(1, 'rgba(15, 23, 42, 0.9)');
    } else {
      bgGrad.addColorStop(0, 'rgba(30, 41, 59, 0.9)');
      bgGrad.addColorStop(1, 'rgba(15, 23, 42, 0.95)');
    }
    ctx.fillStyle = bgGrad;
    ctx.beginPath();
    ctx.arc(node.x, node.y, r, 0, Math.PI * 2);
    ctx.fill();

    // Border — ripple nodes get a dashed border
    if (node.isRipple) {
      ctx.setLineDash([4, 3]);
      ctx.strokeStyle = isHovered ? 'rgba(6, 182, 212, 0.9)' : 'rgba(6, 182, 212, 0.45)';
    } else {
      ctx.setLineDash([]);
      ctx.strokeStyle = isHovered
        ? 'rgba(129, 140, 248, 0.9)'
        : node.isCenter
          ? 'rgba(99, 102, 241, 0.7)'
          : 'rgba(71, 85, 105, 0.6)';
    }
    ctx.lineWidth = isHovered ? 2.5 : node.isCenter ? 2 : 1.5;
    ctx.stroke();
    ctx.setLineDash([]);

    // Ticker text
    ctx.fillStyle = node.isCenter ? '#c7d2fe' : (node.isRipple ? '#67e8f9' : '#e2e8f0');
    ctx.font = `700 ${node.isCenter ? 13 : (node.isRipple ? 10 : 11)}px Inter, sans-serif`;
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText(node.ticker, node.x, node.y - (node.isCenter ? 6 : 4));

    // Company name (truncated)
    ctx.fillStyle = '#94a3b8';
    ctx.font = `400 ${node.isCenter ? 9 : (node.isRipple ? 7 : 8)}px Inter, sans-serif`;
    const maxW = r * 1.8;
    let displayName = node.name;
    if (ctx.measureText(displayName).width > maxW) {
      while (ctx.measureText(displayName + '…').width > maxW && displayName.length > 3) {
        displayName = displayName.slice(0, -1);
      }
      displayName += '…';
    }
    ctx.fillText(displayName, node.x, node.y + (node.isCenter ? 8 : 6));

    ctx.restore();
  }

  // --- Convert screen coords to world coords (accounting for pan and zoom) ---

  private screenToWorld(screenX: number, screenY: number): { x: number; y: number } {
    return {
      x: (screenX - this.panX) / this.zoom,
      y: (screenY - this.panY) / this.zoom
    };
  }

  // --- Mouse / Touch Interaction ---

  onCanvasMouseMove(event: MouseEvent) {
    const rect = this.canvas.getBoundingClientRect();
    const screenX = event.clientX - rect.left;
    const screenY = event.clientY - rect.top;
    this.mouseX = screenX;
    this.mouseY = screenY;

    // Middle-mouse panning
    if (this.isPanning) {
      this.panX = this.panOriginX + (screenX - this.panStartX);
      this.panY = this.panOriginY + (screenY - this.panStartY);
      this.tooltipVisible = false;
      if (!this.simulationRunning) this.render();
      else this.ensureAnimating();
      return;
    }

    const world = this.screenToWorld(screenX, screenY);

    // Node dragging
    if (this.isDragging && this.draggedNode) {
      const dx = screenX - this.dragStartX;
      const dy = screenY - this.dragStartY;
      if (Math.abs(dx) > 4 || Math.abs(dy) > 4) {
        this.dragMoved = true;
      }
      this.draggedNode.x = world.x;
      this.draggedNode.y = world.y;
      this.draggedNode.vx = 0;
      this.draggedNode.vy = 0;
      this.simulationAlpha = Math.max(this.simulationAlpha, 0.1);
      this.simulationRunning = true;
      this.ensureAnimating();
      return;
    }

    // Hit test nodes (in world coordinates)
    let foundNode: GraphNode | null = null;
    for (const node of this.nodes) {
      const dx = world.x - node.x;
      const dy = world.y - node.y;
      if (dx * dx + dy * dy < (node.radius + 5) * (node.radius + 5)) {
        foundNode = node;
        break;
      }
    }

    // Hit test edges
    let foundEdge: GraphEdge | null = null;
    if (!foundNode) {
      for (const edge of this.edges) {
        if (edge.opacity <= 0) continue;
        if (this.isNearEdge(edge, world.x, world.y, 10)) {
          foundEdge = edge;
          break;
        }
      }
    }

    const prevNode = this.hoveredNode;
    const prevEdge = this.hoveredEdge;
    this.hoveredNode = foundNode;
    this.hoveredEdge = foundEdge;

    // Update tooltip
    if (foundNode) {
      this.canvas.style.cursor = 'pointer';
      this.ngZone.run(() => {
        this.tooltipVisible = true;
        this.tooltipX = event.clientX;
        this.tooltipY = event.clientY;
        this.tooltipTicker = foundNode!.ticker;
        this.tooltipName = foundNode!.name;
        this.tooltipDescription = foundNode!.description;
        this.tooltipSector = foundNode!.sector;
        this.tooltipRelationship = '';
        this.tooltipRelLabel = '';
      });
    } else if (foundEdge) {
      this.canvas.style.cursor = 'default';
      this.ngZone.run(() => {
        this.tooltipVisible = true;
        this.tooltipX = event.clientX;
        this.tooltipY = event.clientY;
        this.tooltipTicker = foundEdge!.source.ticker + ' ↔ ' + foundEdge!.target.ticker;
        this.tooltipName = foundEdge!.label;
        this.tooltipDescription = '';
        this.tooltipSector = '';
        this.tooltipRelationship = foundEdge!.relationship;
        this.tooltipRelLabel = foundEdge!.label;
      });
    } else {
      this.canvas.style.cursor = this.isPanning ? 'grabbing' : 'default';
      if (this.tooltipVisible) {
        this.ngZone.run(() => this.tooltipVisible = false);
      }
    }

    // Redraw if hover state changed
    if (prevNode !== this.hoveredNode || prevEdge !== this.hoveredEdge) {
      if (!this.simulationRunning) {
        this.render();
      }
    }
  }

  onCanvasMouseDown(event: MouseEvent) {
    // Middle mouse button (button === 1) → pan
    if (event.button === 1) {
      event.preventDefault();
      this.isPanning = true;
      const rect = this.canvas.getBoundingClientRect();
      this.panStartX = event.clientX - rect.left;
      this.panStartY = event.clientY - rect.top;
      this.panOriginX = this.panX;
      this.panOriginY = this.panY;
      this.canvas.style.cursor = 'grabbing';
      return;
    }

    // Left mouse button → drag node
    if (event.button === 0 && this.hoveredNode) {
      const rect = this.canvas.getBoundingClientRect();
      this.draggedNode = this.hoveredNode;
      this.isDragging = true;
      this.dragMoved = false;
      this.dragStartX = event.clientX - rect.left;
      this.dragStartY = event.clientY - rect.top;
      event.preventDefault();
    }
  }

  onCanvasMouseUp(event: MouseEvent) {
    // End panning
    if (event.button === 1) {
      this.isPanning = false;
      this.canvas.style.cursor = 'default';
      return;
    }

    // End dragging — only navigate if it was a clean click (no movement)
    if (event.button === 0) {
      if (this.isDragging && this.draggedNode && !this.dragMoved) {
        // This was a click, not a drag → navigate
        const ticker = this.draggedNode.ticker;
        this.isDragging = false;
        this.draggedNode = null;
        this.dragMoved = false;
        this.router.navigate(['/dashboard', ticker]);
        return;
      }
      this.isDragging = false;
      this.draggedNode = null;
      this.dragMoved = false;
    }
  }

  onCanvasClick(event: MouseEvent) {
    // Click handling is now done in mouseUp to properly distinguish drag vs click
    // This handler is kept empty to prevent double navigation
  }

  onCanvasMouseLeave() {
    this.hoveredNode = null;
    this.hoveredEdge = null;
    this.isPanning = false;
    this.isDragging = false;
    this.draggedNode = null;
    this.dragMoved = false;
    this.tooltipVisible = false;
    this.canvas.style.cursor = 'default';
    if (!this.simulationRunning) {
      this.render();
    }
  }

  onCanvasWheel(event: WheelEvent) {
    event.preventDefault();

    const rect = this.canvas.getBoundingClientRect();
    const screenX = event.clientX - rect.left;
    const screenY = event.clientY - rect.top;

    const zoomSensitivity = 0.001;
    const delta = -event.deltaY * zoomSensitivity;

    const oldZoom = this.zoom;
    this.zoom = Math.max(0.1, Math.min(this.zoom * (1 + delta), 4));

    // Adjust pan to zoom into the mouse position
    const worldX = (screenX - this.panX) / oldZoom;
    const worldY = (screenY - this.panY) / oldZoom;

    this.panX = screenX - worldX * this.zoom;
    this.panY = screenY - worldY * this.zoom;

    if (!this.simulationRunning) {
      this.render();
    } else {
      this.ensureAnimating();
    }
  }

  // Prevent context menu on middle-click
  onCanvasContextMenu(event: MouseEvent) {
    if (event.button === 1) {
      event.preventDefault();
    }
  }

  private isNearEdge(edge: GraphEdge, mx: number, my: number, threshold: number): boolean {
    const steps = 20;
    const sx = edge.source.x, sy = edge.source.y;
    const tx = edge.target.x, ty = edge.target.y;
    const midx = (sx + tx) / 2, midy = (sy + ty) / 2;
    const dx = tx - sx, dy = ty - sy;
    const len = Math.sqrt(dx * dx + dy * dy) || 1;
    const nx = -dy / len, ny = dx / len;
    const offset = len * 0.15;
    const cpx = midx + nx * offset, cpy = midy + ny * offset;

    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      const px = (1 - t) * (1 - t) * sx + 2 * (1 - t) * t * cpx + t * t * tx;
      const py = (1 - t) * (1 - t) * sy + 2 * (1 - t) * t * cpy + t * t * ty;
      const ddx = mx - px, ddy = my - py;
      if (ddx * ddx + ddy * ddy < threshold * threshold) return true;
    }
    return false;
  }

  // Legend data
  get legendItems() {
    if (!this.networkData) return [];
    const types = new Set(this.networkData.edges.map(e => e.relationship));
    return Array.from(types).map(t => ({
      type: t,
      color: RELATIONSHIP_COLORS[t] || '#64748b',
      icon: RELATIONSHIP_ICONS[t] || '🔗',
      label: t === 'ripple' ? 'Ripple Effect' : t.charAt(0).toUpperCase() + t.slice(1)
    }));
  }

  goHome() {
    this.router.navigate(['/']);
  }
}
