\# SIH26106 Agent Context



\## 1. Purpose



This document defines the responsibilities, ownership boundaries, and collaboration rules for AI coding agents working on SIH26106.



The project uses two primary coding agents:



\- Agy-CLI

\- Codex-CLI



The user is the final authority for architectural and cross-agent decisions.



\---



\## 2. Repository Ownership



\### Agy-CLI



Primary owner:



```text

frontend/



Agy is responsible for:



React

TypeScript

Vite

UI components

UX

visual design implementation

frontend state

frontend API integration

frontend tests

Codex-CLI



Primary owner:



backend/



Codex is responsible for:



Go

Chi

REST API

email parsing

forensic analysis

analyzers

risk engine

database integration

Redis integration

evidence

audit

backend tests

3\. Absolute Ownership Boundary

Agy MUST NOT modify backend/.

Codex MUST NOT modify frontend/.



This restriction applies even when modifying the other side appears:



convenient

necessary

cleaner

faster

easier

architecturally preferable



An agent must not cross the boundary autonomously.



If the other side needs a change:



Identify the required change.

Explain the reason.

Stop.

Wait for explicit user authorization.



Never silently modify the other agent's code.



4\. Shared Documentation



Both agents may read:



docs/



Important shared documents include:



architecture.md

api-contract.md

data-model.md

analysis-pipeline.md

development-rules.md

ui-ux-design-system.md

agent-context.md

agy-role.md

codex-role.md



These documents define the common understanding of the system.



5\. API Contract Boundary



The API contract is the formal boundary between frontend and backend.



Agy consumes the API.



Codex implements the API.



Neither agent may silently change the API contract to suit its own implementation.



When an API change is required:



Identify the change.

Update docs/api-contract.md.

Implement the backend side.

Update the frontend side.

Test the integration.



The API contract must remain internally consistent.



6\. Feature Development Model



Features are developed as vertical slices.



Example:



Feature

&#x20; ↓

API/Data Contract

&#x20; ↓

Codex Backend

&#x20; +

Agy Frontend

&#x20; ↓

Integration

&#x20; ↓

Testing

&#x20; ↓

Commit



Do not build the entire frontend before the backend exists.



Do not build the entire backend before the frontend exists.



7\. MVP Philosophy



The objective is a clean, functional MVP.



Prefer:



simple architecture

clear responsibilities

working features

good UX

testability

extensibility



Avoid:



speculative infrastructure

microservices

unnecessary abstractions

premature optimization

features that are not needed for the MVP

8\. Design Specification



docs/ui-ux-design-system.md defines the intended visual language and UX principles.



It is a design specification, not a requirement to implement every component immediately.



Implementation should proceed incrementally according to the active feature.



9\. Evidence and Forensic Integrity



The system must clearly distinguish:



OBSERVED

ENRICHED

INFERRED

AI-ASSESSED



Do not present inferred or AI-generated conclusions as observed facts.



Geolocation is an estimate.



AI attribution is an assessment.



Conclusions should be traceable to supporting evidence.



10\. Agent Behaviour



Every agent must:



inspect the repository before editing

read relevant documentation

understand existing code before changing it

minimize unrelated modifications

avoid speculative features

run appropriate tests

run build checks

report failures honestly



Agents must not:



rewrite the architecture without authorization

modify the other agent's codebase

invent undocumented APIs

silently break existing contracts

claim tests passed when they were not run

11\. User Authority



The user has final authority over:



architecture

API contract

data model

design decisions

cross-agent changes

dependency decisions

feature scope

