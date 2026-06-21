import { Component, Input } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-loading-spinner',
  standalone: true,
  imports: [CommonModule],
  template: `
    <div class="loading-section animate-fade-in">
      <div class="loading-spinner">
        <div class="spinner-ring"></div>
        <div class="spinner-ring delay-1"></div>
        <div class="spinner-ring delay-2"></div>
      </div>
      <p class="loading-text" *ngIf="text">{{ text }}</p>
      <p class="loading-subtext" *ngIf="subtext">{{ subtext }}</p>
    </div>
  `,
  styleUrls: ['./loading-spinner.component.css']
})
export class LoadingSpinnerComponent {
  @Input() text: string = 'Loading...';
  @Input() subtext: string = '';
}
