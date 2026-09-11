import React from 'react';
import { ArrowRight, CheckCircle2, ChevronDown, ShieldCheck, Sparkles, Upload } from 'lucide-react';
import './LandingPage.css';

interface Props {
  onGetStarted: () => void;
  onSignIn: () => void;
}

export const LandingPage: React.FC<Props> = ({ onGetStarted, onSignIn }) => (
  <main className="landing-page">
    <nav className="landing-nav" aria-label="Primary navigation">
      <button type="button" className="landing-brand" onClick={() => window.scrollTo({ top: 0, behavior: 'smooth' })}>
        <span className="landing-brand-mark">S</span>
        <span>SIH<span>26106</span></span>
      </button>
      <div className="landing-links">
        <a href="#capabilities">Capabilities <ChevronDown size={14} /></a>
        <a href="#workflow">Workflow</a>
        <a href="#trust">Trust & safety</a>
        <a href="#about">About</a>
      </div>
      <div className="landing-actions">
        <button type="button" className="landing-sign-in" onClick={onSignIn}>Sign in</button>
        <button type="button" className="landing-outline-button" onClick={onGetStarted}>Get started <ArrowRight size={16} /></button>
      </div>
    </nav>

    <section className="landing-hero">
      <div className="landing-hero-copy">
        <span className="landing-kicker"><Sparkles size={15} /> Forensic email intelligence</span>
        <h1>Find the signal.<br /><em>Act with confidence.</em></h1>
        <p>Investigate suspicious email with evidence-backed parsing, AI-assisted assessment, and a clear chain of forensic context.</p>
        <div className="landing-hero-actions">
          <button type="button" className="landing-primary-button" onClick={onGetStarted}>Open investigation <ArrowRight size={18} /></button>
          <a className="landing-text-link" href="#workflow">See how it works <ArrowRight size={16} /></a>
        </div>
        <div className="landing-proof"><CheckCircle2 size={16} /> Built for careful analysis, not automatic conclusions</div>
      </div>
      <div className="landing-visual" aria-label="Investigation workspace preview">
        <div className="visual-window">
          <div className="visual-window-bar"><span /><span /><span /><strong>Investigation workspace</strong><small>LIVE</small></div>
          <div className="visual-body">
            <div className="visual-sidebar"><b>CASE FILE</b><i className="active">Overview</i><i>Headers & auth</i><i>Evidence vault</i><i>Entity graph</i><i>Report</i></div>
            <div className="visual-content"><div className="visual-label">THREAT ASSESSMENT</div><h3>Payment request anomaly</h3><div className="visual-score"><strong>72</strong><span>HIGH RISK</span></div><div className="visual-lines"><i /><i /><i /><i /><i /></div><div className="visual-cards"><span>SPF fail</span><span>URL observed</span><span>AI assessed</span></div></div>
          </div>
        </div>
        <div className="visual-glow" />
      </div>
    </section>

    <section className="landing-capabilities" id="capabilities">
      <div><span className="landing-kicker">One workspace, clear answers</span><h2>From raw email to defensible insight.</h2></div>
      <div className="capability-grid">
        <article><ShieldCheck size={22} /><h3>Preserve the evidence</h3><p>Capture headers, MIME structure, indicators, and provenance without altering the original artifact.</p></article>
        <article><Sparkles size={22} /><h3>Assess with AI</h3><p>Use AI as an evaluated assessment alongside deterministic signals, never as ground truth.</p></article>
        <article><Upload size={22} /><h3>Report clearly</h3><p>Move from upload to a structured investigation with evidence, timeline, graph, and report views.</p></article>
      </div>
    </section>

    <section className="landing-workflow" id="workflow">
      <div className="landing-section-heading">
        <span className="landing-kicker">Workflow</span>
        <h2>Every conclusion keeps a path back to the artifact.</h2>
        <p>SIH26106 combines deterministic email forensics with a bounded AI assessment so analysts can move quickly without losing context.</p>
      </div>
      <div className="workflow-grid">
        {[
          ['01', 'Ingest', 'Preserve the original .eml and create a traceable investigation case.'],
          ['02', 'Extract', 'Normalize headers, MIME parts, indicators, authentication results, and attachments.'],
          ['03', 'Assess', 'Combine deterministic risk signals with a confidence-scored Gemini assessment.'],
          ['04', 'Explain', 'Review evidence, relay timeline, entity graph, enrichment, and a safe report.'],
        ].map(([number, title, description]) => (
          <article className="workflow-card" key={number}>
            <span className="workflow-number">{number}</span>
            <h3>{title}</h3>
            <p>{description}</p>
          </article>
        ))}
      </div>
    </section>

    <section className="landing-trust" id="trust">
      <div>
        <span className="landing-kicker"><ShieldCheck size={15} /> Trust & safety</span>
        <h2>AI-assisted, evidence-first, safe by default.</h2>
      </div>
      <div className="trust-points">
        <div><strong>01</strong><span>Attachments are never executed. Only metadata and hashes are inspected.</span></div>
        <div><strong>02</strong><span>URLs are recorded as evidence but are not automatically visited.</span></div>
        <div><strong>03</strong><span>Observed, inferred, enriched, and AI-assessed facts remain visibly distinct.</span></div>
      </div>
    </section>

    <section className="landing-about" id="about">
      <div className="about-copy">
        <span className="landing-kicker">About SIH26106</span>
        <h2>A focused investigation console for suspicious email.</h2>
        <p>Built for analysts who need one defensible workspace instead of a collection of disconnected tools. The platform is intentionally modular, explainable, and ready for controlled feedback-driven learning.</p>
        <button type="button" className="landing-primary-button" onClick={onGetStarted}>Start an investigation <ArrowRight size={18} /></button>
      </div>
      <div className="demo-case-card" aria-label="Example investigation data">
        <div className="demo-case-header"><span>DEMO CASE / PAYMENT_REDIRECT_014</span><span className="demo-status">PARTIAL</span></div>
        <h3>Supplier account update request</h3>
        <div className="demo-metrics">
          <div><strong>72</strong><span>risk score</span></div>
          <div><strong>08</strong><span>evidence items</span></div>
          <div><strong>03</strong><span>relay hops</span></div>
        </div>
        <div className="demo-tags"><span>DMARC fail</span><span>URL observed</span><span>AI assessed</span></div>
      </div>
    </section>
  </main>
);
