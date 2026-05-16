# Zora AI – Refined System Description

## Overview

Zora AI is an assistant powered by a Large Language Model (LLM). The LLM acts as the reasoning engine that makes decisions to solve user tasks.

The system consists of:

* **Tools** — Executable capabilities that Zora can invoke.
* **Knowledge** — Embedded information provided by users, such as documents, PDFs, images, books, and behavioral data.

Zora AI uses the LLM to:

1. Understand the user’s task.
2. Determine which tools and knowledge are required.
3. Execute necessary actions.
4. Return the final result.

---

## Core Repositories

1. **zora-core** — Main orchestration system.
2. **zora-knowledge** — Knowledge ingestion and processing system.
3. **zora-mcp-server** — Tool execution layer.

---

## Multi-User Architecture & Personalization

Zora AI is designed to be used by multiple users across platforms such as **WAHA** and **Discord**.

Therefore, the system must support:

### 1. User-Specific Knowledge

* Each user has isolated personal knowledge.
* Personal documents, preferences, behavior history, and learned patterns are stored per user.
* Retrieval must always be scoped to the requesting user.
* Personalization must not leak across users.

This enables:

* User behavior learning.
* Adaptive responses.
* Context-aware assistance.

---

### 2. Admin-Level General Knowledge

* Administrators can add global knowledge.
* General knowledge is shared across all users.
* Used for system-wide information, policies, base instructions, or shared domain expertise.

Retrieval priority should follow:

1. User-specific knowledge.
2. General (admin) knowledge.
3. Tools (if knowledge is not sufficient).

---

## How Zora AI Solves Tasks

When a user sends a request (via BOT, WAHA, or CLI), for example:

```
!zora generate Go code for an LRU and LFU cache
```

The system should:

1. Analyze the task.
2. Retrieve only relevant tools and knowledge (scoped by user).
3. Send filtered context to the LLM.
4. Generate a step-by-step plan.
5. Execute required tools if needed.
6. Return the final response.

The system must avoid loading all tools or all knowledge into the context window.

---

## Current Problem

### Vector-Based Retrieval Limitations

The system currently uses embeddings (pgvector) to search tools and knowledge.

Issues:

* Some tool descriptions are too generic.
* Embedding similarity may return unrelated tools.
* Too many results are retrieved.
* Context window becomes overloaded.
* Precision decreases.

---

## Proposed Improvement: Hybrid Tag-Based System

### 1. LLM-Based Tag Generation

During tool or knowledge creation:

* Use an LLM to generate structured tags.
* Store tags in the database.

---

### 2. Task-Based Tag Matching

When a user submits a request:

* Extract intent and tags.
* Match with stored tags.
* Retrieve only highly relevant tools and knowledge.

This reduces context overload and improves precision.

---

## Knowledge Expectations

### PDF Processing (Background Workflow)

When a user uploads a PDF:

For each page:

1. Send page content to LLM.
2. Generate summary.
3. Generate tags.
4. Store summary, page number, object key, and metadata.

This process must run asynchronously via zora-core and zora-knowledge.

---

### Smart Retrieval Strategy

#### Case A — Information Exists in Summary

* Use stored summary.

#### Case B — Summary Insufficient

* Retrieve full original page.
* Send complete content to LLM.

---

### Image Handling

* Store image path.
* Generate image summary.
* Use summary for search.
* Load full image only when necessary.

---

## Tool vs Knowledge Decision Logic

* If relevant knowledge exists → use it.
* If not → use appropriate tools.

This prevents unnecessary execution and reduces cost.

---

## Architectural Principles

* Modular design.
* Clear separation of concerns.
* Hybrid retrieval (tags + embeddings).
* User-scoped knowledge isolation.
* Admin-level global knowledge support.
* Context budget control.
* Background processing.
* Cost-aware LLM usage.
* Scalable container-based infrastructure.

---

## Long-Term Vision

Zora AI should become:

* Tool-aware.
* Knowledge-driven.
* Multi-user safe.
* Personalization-enabled.
* Context-efficient.
* Self-improving.
* Robust and scalable.

The system must always retrieve only what is necessary and ensure proper isolation between users while allowing controlled global knowledge management.
