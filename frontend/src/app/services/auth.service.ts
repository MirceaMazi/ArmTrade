import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable, BehaviorSubject } from 'rxjs';
import { tap } from 'rxjs/operators';

export interface AuthResponse {
  token: string;
  username: string;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private apiUrl = 'http://localhost:8080/api/auth';
  private loggedIn = new BehaviorSubject<boolean>(this.hasToken());
  private username = new BehaviorSubject<string>(this.getStoredUsername());

  isLoggedIn$ = this.loggedIn.asObservable();
  username$ = this.username.asObservable();

  constructor(private http: HttpClient) {}

  register(username: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.apiUrl}/register`, { username, password }).pipe(
      tap(res => this.setSession(res))
    );
  }

  login(username: string, password: string): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.apiUrl}/login`, { username, password }).pipe(
      tap(res => this.setSession(res))
    );
  }

  logout(): void {
    localStorage.removeItem('token');
    localStorage.removeItem('username');
    this.loggedIn.next(false);
    this.username.next('');
  }

  getToken(): string | null {
    return localStorage.getItem('token');
  }

  isLoggedIn(): boolean {
    return this.hasToken();
  }

  getUsername(): string {
    return localStorage.getItem('username') || '';
  }

  private setSession(res: AuthResponse): void {
    localStorage.setItem('token', res.token);
    localStorage.setItem('username', res.username);
    this.loggedIn.next(true);
    this.username.next(res.username);
  }

  private hasToken(): boolean {
    return !!localStorage.getItem('token');
  }

  private getStoredUsername(): string {
    return localStorage.getItem('username') || '';
  }
}
