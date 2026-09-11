import React, { useState } from 'react';
import { ArrowLeft, ArrowRight, Eye, EyeOff, ShieldCheck } from 'lucide-react';
import './LoginPage.css';

interface Props { onBack: () => void; onAuthenticated: () => void; }

export const LoginPage: React.FC<Props> = ({ onBack, onAuthenticated }) => {
  const [showPassword, setShowPassword] = useState(false);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    if (!email.trim() || !password.trim()) { setError('Enter your email and password to continue.'); return; }
    setError('');
    onAuthenticated();
  };
  return (
    <main className="login-page">
      <button type="button" className="login-back" onClick={onBack}><ArrowLeft size={16} /> Back to home</button>
      <section className="login-card">
        <div className="login-brand"><span className="landing-brand-mark">S</span><span>SIH<span>26106</span></span></div>
        <ShieldCheck className="login-shield" size={30} />
        <h1>Welcome back</h1>
        <p className="login-subtitle">Sign in to continue to your forensic workspace</p>
        <form onSubmit={submit}>
          <label>Email address<input type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="you@organization.com" autoComplete="email" /></label>
          <label>Password<div className="password-field"><input type={showPassword ? 'text' : 'password'} value={password} onChange={(event) => setPassword(event.target.value)} placeholder="Enter your password" autoComplete="current-password" /><button type="button" onClick={() => setShowPassword(!showPassword)} aria-label={showPassword ? 'Hide password' : 'Show password'}>{showPassword ? <EyeOff size={17} /> : <Eye size={17} />}</button></div></label>
          <div className="login-options"><label className="remember"><input type="checkbox" /> Remember me</label><button type="button" className="login-forgot" onClick={() => { setNotice('Password recovery is available once identity services are connected.'); setError(''); }}>Forgot password?</button></div>
          {error && <p className="login-error" role="alert">{error}</p>}
          {notice && <p className="login-notice" role="status">{notice}</p>}
          <button type="submit" className="login-submit">Sign in <ArrowRight size={17} /></button>
        </form>
        <p className="login-footer">New to SIH26106? <button type="button" onClick={onBack}>Explore the platform</button></p>
        <small className="login-note">Demo authentication is enabled for this MVP. Backend account management can be connected when identity services are added.</small>
      </section>
    </main>
  );
};
