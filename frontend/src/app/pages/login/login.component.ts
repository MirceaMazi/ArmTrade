import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { ButtonModule } from 'primeng/button';
import { InputTextModule } from 'primeng/inputtext';
import { AuthService } from '../../services/auth.service';

@Component({
  selector: 'app-login',
  standalone: true,
  imports: [CommonModule, FormsModule, ButtonModule, InputTextModule],
  templateUrl: './login.component.html',
  styleUrl: './login.component.css'
})
export class LoginComponent {
  isRegisterMode = false;
  username = '';
  password = '';
  error = '';
  loading = false;

  constructor(private authService: AuthService, private router: Router) {}

  toggleMode() {
    this.isRegisterMode = !this.isRegisterMode;
    this.error = '';
  }

  submit() {
    if (!this.username.trim() || !this.password.trim()) {
      this.error = 'Please fill in all fields';
      return;
    }

    this.loading = true;
    this.error = '';

    const action = this.isRegisterMode
      ? this.authService.register(this.username, this.password)
      : this.authService.login(this.username, this.password);

    action.subscribe({
      next: () => {
        this.loading = false;
        this.router.navigate(['/']);
      },
      error: (err) => {
        this.loading = false;
        this.error = err.error?.error || 'Something went wrong. Please try again.';
      }
    });
  }

  goBack() {
    this.router.navigate(['/']);
  }
}
