import { Component } from '@angular/core';
import { CommonModule, Location } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { SkeletonModule } from 'primeng/skeleton';
import { TagModule } from 'primeng/tag';
import { StockService } from '../../services/stock.service';
import * as mammoth from 'mammoth';
import * as pdfjsLib from 'pdfjs-dist';

// Point pdf.js to the worker script
pdfjsLib.GlobalWorkerOptions.workerSrc = 'https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.worker.min.js';

@Component({
  selector: 'app-earnings',
  standalone: true,
  imports: [CommonModule, FormsModule, ButtonModule, SkeletonModule, TagModule],
  templateUrl: './earnings.component.html',
  styleUrl: './earnings.component.css'
})
export class EarningsComponent {
  transcript = '';
  ticker = '';
  loading = false;
  summary: any = null;

  selectedFileName = '';
  parsingFile = false;
  fileError = '';

  constructor(private stockService: StockService, private router: Router, private location: Location) {}

  async onFileSelected(event: any) {
    const file = event.target.files[0];
    if (!file) return;

    this.selectedFileName = file.name;
    this.parsingFile = true;
    this.fileError = '';
    this.transcript = '';

    try {
      if (file.name.toLowerCase().endsWith('.pdf')) {
        this.transcript = await this.extractTextFromPDF(file);
      } else if (file.name.toLowerCase().endsWith('.docx')) {
        this.transcript = await this.extractTextFromDocx(file);
      } else if (file.name.toLowerCase().endsWith('.html') || file.name.toLowerCase().endsWith('.htm')) {
        this.transcript = await this.extractTextFromHTML(file);
      } else {
        this.fileError = 'Unsupported file format. Please upload PDF, DOCX, or HTML.';
      }
    } catch (err: any) {
      console.error(err);
      this.fileError = 'Error parsing file: ' + err.message;
    } finally {
      this.parsingFile = false;
    }
  }

  private async extractTextFromPDF(file: File): Promise<string> {
    const arrayBuffer = await file.arrayBuffer();
    const pdf = await pdfjsLib.getDocument({ data: arrayBuffer }).promise;
    let fullText = '';
    
    for (let i = 1; i <= pdf.numPages; i++) {
      const page = await pdf.getPage(i);
      const textContent = await page.getTextContent();
      const pageText = textContent.items.map((item: any) => item.str).join(' ');
      fullText += pageText + '\n\n';
    }
    return fullText;
  }

  private async extractTextFromDocx(file: File): Promise<string> {
    const arrayBuffer = await file.arrayBuffer();
    const result = await mammoth.extractRawText({ arrayBuffer: arrayBuffer });
    return result.value;
  }

  private async extractTextFromHTML(file: File): Promise<string> {
    const content = await file.text();
    const parser = new DOMParser();
    const htmlDoc = parser.parseFromString(content, 'text/html');
    
    // Remove script and style elements
    const scripts = htmlDoc.querySelectorAll('script, style');
    scripts.forEach(script => script.remove());
    
    // Get text content and normalize whitespace
    const text = htmlDoc.body.textContent || '';
    return text.replace(/\s+/g, ' ').trim();
  }

  summarize() {
    if (!this.transcript.trim()) return;
    this.loading = true;
    this.summary = null;

    this.stockService.summarizeEarnings(this.transcript, this.ticker).subscribe({
      next: (res) => {
        this.summary = res;
        this.loading = false;
      },
      error: () => this.loading = false
    });
  }

  getSentimentSeverity(sentiment: string): 'success' | 'danger' | 'secondary' {
    if (sentiment === 'Bullish') return 'success';
    if (sentiment === 'Bearish') return 'danger';
    return 'secondary';
  }

  goBack() {
    this.location.back();
  }
}
