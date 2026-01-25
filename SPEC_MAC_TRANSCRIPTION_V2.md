# Mac Screen Transcription App - Technical Specification v2.0

## Project Codename: "Scribe"

**Version:** 2.0.0
**Status:** Final Draft
**Date:** 2025-01-25
**Document Type:** Spec-Driven Development Reference

---

## Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [System Architecture](#2-system-architecture)
3. [Core Feature Specifications](#3-core-feature-specifications)
4. [Technology Stack](#4-technology-stack)
5. [Data Flow & Processing Pipeline](#5-data-flow--processing-pipeline)
6. [Speaker Learning System](#6-speaker-learning-system)
7. [Audio Quality & Constraints](#7-audio-quality--constraints)
8. [Edge Cases & Error Handling](#8-edge-cases--error-handling)
9. [Data Models & Storage](#9-data-models--storage)
10. [API Specification](#10-api-specification)
11. [User Interface Specification](#11-user-interface-specification)
12. [Performance Benchmarks](#12-performance-benchmarks)
13. [Security & Privacy](#13-security--privacy)
14. [Development Phases](#14-development-phases)
15. [Acceptance Criteria](#15-acceptance-criteria)

---

## 1. Executive Summary

### 1.1 Product Vision

Build an Otter.ai-like transcription application for macOS that:
- Captures audio from screen recordings or live system audio
- Transcribes speech with high accuracy using Whisper
- Performs speaker diarization (identifying distinct speakers)
- Learns speaker identities from user-provided labels
- Automatically identifies known speakers in future recordings
- Processes all data locally on the user's Mac for privacy

### 1.2 Core User Workflow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           USER WORKFLOW                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  1. CAPTURE/UPLOAD                                                           │
│     ┌─────────────┐      ┌─────────────┐      ┌─────────────┐               │
│     │   Screen    │  OR  │   Audio     │  OR  │    Live     │               │
│     │  Recording  │      │   Upload    │      │ Microphone  │               │
│     └──────┬──────┘      └──────┬──────┘      └──────┬──────┘               │
│            └─────────────────────┴─────────────────────┘                     │
│                                  │                                           │
│                                  ▼                                           │
│  2. AUTOMATIC PROCESSING                                                     │
│     ┌─────────────────────────────────────────────────────────┐             │
│     │  Diarization → Transcription → Speaker Matching         │             │
│     │  "Speaker A", "Speaker B", "Speaker C"                  │             │
│     │                                                          │             │
│     │  IF known speaker detected (>75% match):                │             │
│     │     → Auto-assign: "Speaker A" → "Bill"                 │             │
│     │  IF possible match (60-75%):                            │             │
│     │     → Suggest: "Speaker B looks like Gill (68%)"        │             │
│     │  IF no match (<60%):                                    │             │
│     │     → Keep temporary: "Speaker C"                       │             │
│     └─────────────────────────────────────────────────────────┘             │
│                                  │                                           │
│                                  ▼                                           │
│  3. MANUAL LABELING (User Action)                                           │
│     ┌─────────────────────────────────────────────────────────┐             │
│     │  User clicks "Speaker C" → Labels as "Alex"             │             │
│     │                                                          │             │
│     │  System Response:                                        │             │
│     │  • Extracts voice embeddings from labeled segments      │             │
│     │  • Creates/updates Alex's speaker profile               │             │
│     │  • Applies label to all Speaker C segments              │             │
│     │  • Re-matches other unidentified speakers               │             │
│     └─────────────────────────────────────────────────────────┘             │
│                                  │                                           │
│                                  ▼                                           │
│  4. FUTURE RECORDINGS                                                        │
│     ┌─────────────────────────────────────────────────────────┐             │
│     │  Alex's voice is automatically recognized               │             │
│     │  Confidence improves with more labeled samples          │             │
│     └─────────────────────────────────────────────────────────┘             │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.3 Key Design Principles

| Principle | Description |
|-----------|-------------|
| **Local-First** | All processing happens on-device; no data leaves Mac without explicit consent |
| **Progressive Learning** | System accuracy improves with each user correction |
| **Confidence-Driven UX** | Auto-assign when confident, suggest when uncertain, ask when unknown |
| **Graceful Degradation** | System works even with poor audio, just with lower confidence |
| **User Control** | Users can always override, correct, or undo automatic decisions |

---

## 2. System Architecture

### 2.1 High-Level Architecture Diagram

```
┌──────────────────────────────────────────────────────────────────────────────────┐
│                        SCRIBE - MAC DESKTOP APPLICATION                           │
│                                                                                   │
│  ┌─────────────────────────────────────────────────────────────────────────────┐ │
│  │                         TAURI 2.0 NATIVE SHELL                               │ │
│  │  ┌───────────────┐  ┌───────────────┐  ┌────────────────────────────────┐   │ │
│  │  │ ScreenCapture │  │  System Tray  │  │     Native File System         │   │ │
│  │  │     Kit       │  │   Controls    │  │        Access                  │   │ │
│  │  └───────┬───────┘  └───────────────┘  └────────────────────────────────┘   │ │
│  └──────────┼──────────────────────────────────────────────────────────────────┘ │
│             │                                                                     │
│  ┌──────────┼──────────────────────────────────────────────────────────────────┐ │
│  │          │              NEXT.JS 15 FRONTEND (WebView)                        │ │
│  │          │                                                                    │ │
│  │  ┌───────▼───────┐  ┌─────────────────┐  ┌─────────────────────────────────┐│ │
│  │  │  Recording    │  │   Transcript    │  │      Speaker Management         ││ │
│  │  │   Manager     │  │     Viewer      │  │   • Profile List                ││ │
│  │  │  • Upload     │  │  • Segments     │  │   • Labeling Modal              ││ │
│  │  │  • Capture    │  │  • Playback     │  │   • Merge/Delete                ││ │
│  │  │  • Progress   │  │  • Search       │  │   • Confidence Display          ││ │
│  │  └───────────────┘  └─────────────────┘  └─────────────────────────────────┘│ │
│  │                                                                              │ │
│  │  State: Zustand + SWR          │          API Client: fetch + WebSocket     │ │
│  └──────────────────────────────────────────────────────────────────────────────┘ │
│                                   │                                               │
│                                   │ REST + WebSocket                              │
│                                   ▼                                               │
│  ┌──────────────────────────────────────────────────────────────────────────────┐ │
│  │                         FASTAPI BACKEND (Python 3.11+)                        │ │
│  │                                                                               │ │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐ │ │
│  │  │                        PROCESSING PIPELINE                               │ │ │
│  │  │                                                                          │ │ │
│  │  │  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────────────────┐  │ │ │
│  │  │  │  Audio   │──▶│   VAD    │──▶│ Diarize  │──▶│     Transcribe       │  │ │ │
│  │  │  │ Extract  │   │ (Silero) │   │(pyannote)│   │     (Whisper)        │  │ │ │
│  │  │  └──────────┘   └──────────┘   └──────────┘   └──────────┬───────────┘  │ │ │
│  │  │                                                          │              │ │ │
│  │  │                                                          ▼              │ │ │
│  │  │  ┌──────────────────────────────────────────────────────────────────┐   │ │ │
│  │  │  │                   SPEAKER LEARNING ENGINE                         │   │ │ │
│  │  │  │                                                                   │   │ │ │
│  │  │  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐   │   │ │ │
│  │  │  │  │  Embedding  │  │   Profile   │  │     Similarity          │   │   │ │ │
│  │  │  │  │  Extractor  │  │   Manager   │  │      Matcher            │   │   │ │ │
│  │  │  │  │ (ECAPA-TDNN)│  │             │  │                         │   │   │ │ │
│  │  │  │  │  192-dim    │  │  Centroid   │  │  >0.75 = auto-assign    │   │   │ │ │
│  │  │  │  │  vectors    │  │  + Samples  │  │  0.60-0.75 = suggest    │   │   │ │ │
│  │  │  │  │             │  │  (max 50)   │  │  <0.60 = new speaker    │   │   │ │ │
│  │  │  │  └─────────────┘  └─────────────┘  └─────────────────────────┘   │   │ │ │
│  │  │  └──────────────────────────────────────────────────────────────────┘   │ │ │
│  │  └─────────────────────────────────────────────────────────────────────────┘ │ │
│  │                                                                               │ │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐   │ │
│  │  │  Export Service │  │  Search (FTS5)  │  │    WebSocket Publisher      │   │ │
│  │  │  TXT/SRT/VTT/   │  │  Full-text on   │  │    Real-time updates        │   │ │
│  │  │  JSON/DOCX      │  │  transcripts    │  │    Progress streaming       │   │ │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘   │ │
│  └──────────────────────────────────────────────────────────────────────────────┘ │
│                                   │                                               │
│                                   ▼                                               │
│  ┌──────────────────────────────────────────────────────────────────────────────┐ │
│  │                         LOCAL STORAGE (SQLite)                                │ │
│  │                                                                               │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────┐  │ │
│  │  │ recordings  │  │  speakers   │  │  segments   │  │ speaker_embeddings  │  │ │
│  │  │             │  │             │  │             │  │                     │  │ │
│  │  │ • id        │  │ • id        │  │ • id        │  │ • id                │  │ │
│  │  │ • title     │  │ • name      │  │ • text      │  │ • speaker_id        │  │ │
│  │  │ • duration  │  │ • centroid  │  │ • start_ms  │  │ • embedding (BLOB)  │  │ │
│  │  │ • status    │  │ • samples   │  │ • end_ms    │  │ • quality_score     │  │ │
│  │  │ • path      │  │ • confidence│  │ • speaker_id│  │ • created_at        │  │ │
│  │  └─────────────┘  └─────────────┘  └─────────────┘  └─────────────────────┘  │ │
│  │                                                                               │ │
│  │  ┌─────────────────────────────┐  ┌─────────────────────────────────────────┐│ │
│  │  │     segments_fts (FTS5)     │  │         speaker_vectors (VSS)           ││ │
│  │  │   Full-text search index    │  │      Vector similarity search           ││ │
│  │  └─────────────────────────────┘  └─────────────────────────────────────────┘│ │
│  └──────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                   │
│  ┌──────────────────────────────────────────────────────────────────────────────┐ │
│  │                         FILE STORAGE                                          │ │
│  │  ~/Library/Application Support/Scribe/                                        │ │
│  │  ├── data/scribe.db              (SQLite database)                           │ │
│  │  ├── uploads/                    (Uploaded audio/video files)                │ │
│  │  ├── models/                     (ML models: ECAPA-TDNN, Whisper)            │ │
│  │  └── exports/                    (Exported transcripts)                      │ │
│  └──────────────────────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

| Component | Responsibility | Technology |
|-----------|---------------|------------|
| **Tauri Shell** | Native macOS integration, permissions, system tray, file access | Rust + Tauri 2.0 |
| **Frontend** | User interface, state management, real-time updates | Next.js 15, React 19, TypeScript |
| **Backend** | Audio processing, ML inference, API endpoints | FastAPI, Python 3.11+ |
| **Processing Pipeline** | VAD, diarization, transcription, speaker matching | pyannote, Whisper, SpeechBrain |
| **Speaker Learning** | Embedding extraction, profile management, similarity matching | ECAPA-TDNN, cosine similarity |
| **Storage** | Recordings, speakers, segments, embeddings, search index | SQLite, FTS5, sqlite-vss |

---

## 3. Core Feature Specifications

### 3.1 Feature: Audio File Upload & Processing

**Description:** User uploads audio/video file for transcription and diarization.

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F1.1 | System accepts MP3, WAV, M4A, MP4, WebM, OGG, FLAC files up to 2GB | P0 |
| F1.2 | Files are validated for minimum audio quality before processing | P0 |
| F1.3 | Processing progress is shown in real-time (0-100%) with stage indicator | P0 |
| F1.4 | User can cancel processing at any time | P1 |
| F1.5 | Failed processing shows actionable error message | P0 |
| F1.6 | Processing resumes from last checkpoint after app restart | P2 |

**Audio Quality Validation:**

```python
MINIMUM_AUDIO_REQUIREMENTS = {
    "sample_rate": 8000,      # Minimum 8kHz (16kHz preferred)
    "bit_depth": 16,          # Minimum 16-bit
    "duration_seconds": 5,    # Minimum 5 seconds
    "max_duration_hours": 4,  # Maximum 4 hours per file
    "snr_threshold_db": 10,   # Minimum 10dB signal-to-noise ratio
}
```

### 3.2 Feature: Speaker Diarization

**Description:** Automatically segment audio by speaker ("who spoke when").

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F2.1 | System identifies distinct speakers as "Speaker A", "Speaker B", etc. | P0 |
| F2.2 | Diarization Error Rate (DER) < 15% on clean audio | P0 |
| F2.3 | System handles 2-10 speakers per recording | P0 |
| F2.4 | Minimum segment duration is 1 second (shorter merged with adjacent) | P1 |
| F2.5 | Speaker changes within 500ms are treated as potential overlap | P1 |
| F2.6 | System provides speaker timeline visualization | P1 |

### 3.3 Feature: Transcription

**Description:** Convert speech to text with timestamps.

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F3.1 | Word Error Rate (WER) < 10% on clean English audio | P0 |
| F3.2 | Supports 99 languages with auto-detection | P0 |
| F3.3 | Provides word-level timestamps when requested | P1 |
| F3.4 | Automatic punctuation and capitalization | P0 |
| F3.5 | Processing speed < 0.3x real-time (1hr audio in <20min) | P0 |
| F3.6 | Option to use local model or cloud API | P1 |

### 3.4 Feature: Manual Speaker Labeling

**Description:** User assigns real names to temporary speaker IDs.

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F4.1 | User can click any segment to open labeling modal | P0 |
| F4.2 | Modal shows audio sample playback for verification | P0 |
| F4.3 | Modal suggests matching existing speakers with confidence scores | P0 |
| F4.4 | User can create new speaker or select existing | P0 |
| F4.5 | Label applies to all segments with same temp_speaker_id | P0 |
| F4.6 | **Minimum 3 segments required for initial profile creation** | P0 |
| F4.7 | System warns if labeled segments have low audio quality | P1 |
| F4.8 | Undo available for last labeling action | P1 |

### 3.5 Feature: Automatic Speaker Recognition

**Description:** System automatically identifies known speakers in new recordings.

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F5.1 | Speakers with >0.75 similarity are auto-assigned (high confidence) | P0 |
| F5.2 | Speakers with 0.60-0.75 similarity are suggested (needs confirmation) | P0 |
| F5.3 | Speakers with <0.60 similarity get new temp ID | P0 |
| F5.4 | **Minimum 3-5 labeled segments required before auto-recognition activates** | P0 |
| F5.5 | Recognition accuracy >90% after 10+ labeled samples | P0 |
| F5.6 | User can override any automatic assignment | P0 |
| F5.7 | System shows confidence percentage for all assignments | P1 |

### 3.6 Feature: Export

**Description:** Export transcripts in various formats.

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F6.1 | Export to TXT with timestamps and speaker names | P0 |
| F6.2 | Export to SRT subtitle format | P0 |
| F6.3 | Export to VTT (WebVTT) format | P1 |
| F6.4 | Export to JSON with full metadata | P0 |
| F6.5 | Export to DOCX with formatting | P2 |
| F6.6 | User can choose to include/exclude timestamps | P1 |
| F6.7 | User can choose to include/exclude speaker names | P1 |

### 3.7 Feature: Search

**Description:** Full-text search across all transcripts.

**Acceptance Criteria:**

| ID | Criterion | Priority |
|----|-----------|----------|
| F7.1 | Search returns results in <200ms for typical queries | P0 |
| F7.2 | Results show matched text with highlighting | P0 |
| F7.3 | Results can be filtered by speaker | P1 |
| F7.4 | Results can be filtered by date range | P1 |
| F7.5 | Clicking result navigates to segment in transcript | P0 |

---

## 4. Technology Stack

### 4.1 Technology Decisions with Justifications

| Layer | Technology | Justification |
|-------|------------|---------------|
| **Desktop Framework** | Tauri 2.0 | Native macOS APIs (ScreenCaptureKit), small bundle (~10MB vs 150MB Electron), Rust performance, WebView for UI |
| **Frontend Framework** | Next.js 15 + React 19 | Modern React features, excellent DX, consistent with existing Agentok codebase |
| **UI Components** | Radix UI + TailwindCSS | Accessible primitives, utility-first CSS, rapid development |
| **State Management** | Zustand + SWR | Lightweight global state, built-in caching and revalidation for API data |
| **Backend Framework** | FastAPI | Async Python, automatic OpenAPI docs, excellent for ML workloads |
| **Database** | SQLite | Embedded, zero-config, excellent for local-first apps, supports FTS5 and extensions |
| **Vector Search** | sqlite-vss | SQLite extension for vector similarity, no separate DB needed |
| **Transcription** | OpenAI Whisper (API) + faster-whisper (local) | Best accuracy, API for quick start, local option for privacy |
| **Diarization** | pyannote.audio 3.1 | State-of-the-art DER, active development, Hugging Face integration |
| **Voice Embeddings** | ECAPA-TDNN (SpeechBrain) | 192-dim embeddings, excellent speaker verification, optimized for short utterances |

### 4.2 macOS Integration

| Feature | API/Framework | macOS Version | Notes |
|---------|--------------|---------------|-------|
| Screen Audio Capture | ScreenCaptureKit (SCStream) | 13.0+ | Captures system audio from any app |
| Microphone Capture | AVCaptureSession | 10.7+ | Standard audio input |
| Screen Recording Permission | TCC Framework | 10.14+ | Required for screen capture |
| Microphone Permission | TCC Framework | 10.14+ | Required for mic input |
| Menu Bar Icon | NSStatusItem (via Tauri) | 10.0+ | Quick access controls |
| Notifications | UserNotifications | 10.14+ | Processing complete alerts |
| File Access | NSOpenPanel, sandbox | 10.0+ | User-selected files only |

### 4.3 ML Model Specifications

| Model | Size | Inference Time | Accuracy | Notes |
|-------|------|----------------|----------|-------|
| **Whisper large-v3** (API) | N/A | ~0.1x RT | WER 5-8% | Best accuracy, requires internet |
| **Whisper large-v3** (local) | 3GB | ~0.5x RT | WER 5-8% | Requires Apple Silicon for reasonable speed |
| **faster-whisper large-v3** | 1.5GB | ~0.2x RT | WER 5-8% | INT8 quantized, 4x faster |
| **pyannote/speaker-diarization-3.1** | 200MB | ~0.3x RT | DER 10-15% | Requires HF token |
| **ECAPA-TDNN** | 80MB | <50ms/segment | EER 0.8% | SpeechBrain pretrained |

---

## 5. Data Flow & Processing Pipeline

### 5.1 Complete Processing Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         AUDIO PROCESSING PIPELINE                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  INPUT                                                                           │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  Audio/Video File OR Screen Capture Stream OR Microphone Input           │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 1: AUDIO EXTRACTION (if video)                                           │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  FFmpeg/pydub → Extract audio track → Convert to WAV 16kHz mono          │   │
│  │                                                                           │   │
│  │  Validation:                                                              │   │
│  │  • Sample rate ≥ 8kHz (resample if needed)                               │   │
│  │  • Duration ≥ 5 seconds                                                  │   │
│  │  • SNR estimate > 10dB (warn if lower)                                   │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 2: VOICE ACTIVITY DETECTION                                              │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  Silero VAD → Identify speech vs silence regions                         │   │
│  │                                                                           │   │
│  │  Output: List of (start_ms, end_ms) speech regions                       │   │
│  │  Purpose: Skip silence, reduce processing time                           │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 3: SPEAKER DIARIZATION                                                   │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  pyannote.audio 3.1 → Segment by speaker                                  │   │
│  │                                                                           │   │
│  │  Input: Full audio file                                                   │   │
│  │  Output: List of (speaker_id, start_ms, end_ms) segments                 │   │
│  │                                                                           │   │
│  │  Post-processing:                                                         │   │
│  │  • Merge segments < 1s with adjacent same-speaker                        │   │
│  │  • Mark segments with speaker changes < 500ms as potential overlap       │   │
│  │  • Assign temporary IDs: "Speaker A", "Speaker B", etc.                  │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 4: TRANSCRIPTION (parallel with diarization alignment)                   │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  Whisper (API or local) → Speech-to-text                                  │   │
│  │                                                                           │   │
│  │  Input: Full audio (or speech regions from VAD)                          │   │
│  │  Output: Text with word-level timestamps                                 │   │
│  │                                                                           │   │
│  │  Configuration:                                                           │   │
│  │  • Language: auto-detect or user-specified                               │   │
│  │  • Model: whisper-large-v3 (API) or faster-whisper (local)              │   │
│  │  • Options: word timestamps, punctuation, no profanity filter           │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 5: ALIGNMENT                                                             │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  Merge transcription with diarization                                     │   │
│  │                                                                           │   │
│  │  For each transcription segment:                                          │   │
│  │    1. Find overlapping diarization segment(s)                            │   │
│  │    2. Assign speaker with maximum overlap                                │   │
│  │    3. Handle edge cases (overlap, gaps)                                  │   │
│  │                                                                           │   │
│  │  Output: Segments with (text, start_ms, end_ms, temp_speaker_id)         │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 6: EMBEDDING EXTRACTION                                                  │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  ECAPA-TDNN → Extract voice embeddings for each speaker                  │   │
│  │                                                                           │   │
│  │  For each unique temp_speaker_id:                                         │   │
│  │    1. Collect all segments ≥ 3 seconds                                   │   │
│  │    2. Select up to 5 best quality segments                               │   │
│  │    3. Extract 192-dim embedding for each                                 │   │
│  │    4. Store embeddings with segment references                           │   │
│  │                                                                           │   │
│  │  Quality scoring:                                                         │   │
│  │  • Duration (longer = better, up to 10s)                                 │   │
│  │  • SNR estimate                                                          │   │
│  │  • No overlap with other speakers                                        │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  STAGE 7: SPEAKER MATCHING                                                      │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  Match extracted embeddings against known speaker profiles                │   │
│  │                                                                           │   │
│  │  For each temp_speaker_id:                                                │   │
│  │    1. Compute average embedding from extracted samples                   │   │
│  │    2. Compare against all known speaker centroids (cosine similarity)    │   │
│  │    3. Apply decision logic:                                              │   │
│  │                                                                           │   │
│  │       ┌─────────────────────────────────────────────────────────────┐    │   │
│  │       │  Similarity ≥ 0.75  →  AUTO-ASSIGN (high confidence)        │    │   │
│  │       │  Similarity 0.60-0.75  →  SUGGEST (needs confirmation)      │    │   │
│  │       │  Similarity < 0.60  →  NEW SPEAKER (keep temp ID)           │    │   │
│  │       └─────────────────────────────────────────────────────────────┘    │   │
│  │                                                                           │   │
│  │    4. Store match results with confidence scores                         │   │
│  └────────────────────────────────────┬─────────────────────────────────────┘   │
│                                       │                                          │
│                                       ▼                                          │
│  OUTPUT                                                                          │
│  ┌──────────────────────────────────────────────────────────────────────────┐   │
│  │  Recording marked as "completed"                                          │   │
│  │  Segments stored with:                                                    │   │
│  │  • text, start_ms, end_ms                                                │   │
│  │  • speaker_id (if matched) OR temp_speaker_id                            │   │
│  │  • speaker_confidence                                                     │   │
│  │  • embedding (for future matching)                                       │   │
│  │  • is_user_verified = false                                              │   │
│  │                                                                           │   │
│  │  WebSocket notification sent to frontend                                 │   │
│  └──────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 5.2 Processing Stages & Timing

| Stage | Operation | Typical Time (1hr audio) | Parallelizable |
|-------|-----------|-------------------------|----------------|
| 1 | Audio extraction | 10-30s | No |
| 2 | VAD | 30-60s | No |
| 3 | Diarization | 10-15 min | Yes (with stage 4) |
| 4 | Transcription | 12-18 min | Yes (with stage 3) |
| 5 | Alignment | 5-10s | No |
| 6 | Embedding extraction | 30-60s | Yes (per speaker) |
| 7 | Speaker matching | <1s | Yes (per speaker) |
| **Total** | | **15-25 min** | |

---

## 6. Speaker Learning System

### 6.1 Minimum Requirements for Reliable Speaker Profiles

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    SPEAKER PROFILE REQUIREMENTS                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  MINIMUM THRESHOLDS FOR INITIAL PROFILE CREATION:                               │
│  ─────────────────────────────────────────────────                              │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Labeled Segments:     ≥ 3 segments (REQUIRED)                          │   │
│  │                        ≥ 5 segments (RECOMMENDED for reliability)       │   │
│  │                                                                          │   │
│  │  Segment Duration:     ≥ 3 seconds per segment (REQUIRED)               │   │
│  │                        ≥ 5 seconds per segment (RECOMMENDED)            │   │
│  │                                                                          │   │
│  │  Total Audio:          ≥ 10 seconds (MINIMUM)                           │   │
│  │                        ≥ 30 seconds (RECOMMENDED)                       │   │
│  │                                                                          │   │
│  │  Audio Quality:        SNR > 10dB (REQUIRED)                            │   │
│  │                        No overlapping speakers (RECOMMENDED)            │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  CONFIDENCE LEVELS BASED ON SAMPLE COUNT:                                       │
│  ────────────────────────────────────────                                       │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Samples     Profile Confidence     Auto-Match Reliability              │   │
│  │  ───────     ──────────────────     ──────────────────────              │   │
│  │  3-4         Low (0.15-0.20)        May produce false matches           │   │
│  │  5-9         Medium (0.25-0.45)     Reasonable for familiar voices      │   │
│  │  10-19       High (0.50-0.95)       Reliable for most cases            │   │
│  │  20+         Very High (1.00)       Highly reliable                     │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  UI FEEDBACK FOR SAMPLE REQUIREMENTS:                                           │
│  ─────────────────────────────────────                                          │
│                                                                                  │
│  When user labels < 3 segments:                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  ⚠️ "Label at least 3 segments (10+ seconds) for reliable recognition" │   │
│  │     Current: 2 segments (6 seconds)                                     │   │
│  │     [Label More Segments] [Create Anyway - May Be Unreliable]           │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 6.2 Speaker Matching Algorithm

```python
class SpeakerMatchingConfig:
    """Configuration for speaker matching thresholds."""

    # Similarity thresholds
    AUTO_ASSIGN_THRESHOLD = 0.75      # Auto-assign without confirmation
    SUGGESTION_THRESHOLD = 0.60       # Suggest match, require confirmation
    NEW_SPEAKER_THRESHOLD = 0.60      # Below this, treat as new speaker

    # Minimum requirements
    MIN_SEGMENTS_FOR_PROFILE = 3      # Minimum labeled segments
    MIN_SEGMENT_DURATION_MS = 3000    # Minimum 3 seconds per segment
    MIN_TOTAL_AUDIO_MS = 10000        # Minimum 10 seconds total
    RECOMMENDED_SEGMENTS = 5          # Recommended for reliability
    RECOMMENDED_SEGMENT_DURATION_MS = 5000

    # Profile management
    MAX_SAMPLES_PER_SPEAKER = 50      # Prevent unbounded growth
    ADAPTATION_RATE = 0.95            # Slow adaptation to prevent drift
    DIVERSITY_PRUNE_THRESHOLD = 0.98  # Remove near-duplicate embeddings

    # Quality thresholds
    MIN_SNR_DB = 10                   # Minimum signal-to-noise ratio
    OVERLAP_PENALTY = 0.5             # Reduce quality score for overlaps


class SpeakerMatcher:
    """Match unknown speakers against known profiles."""

    def __init__(self, profiles: List[SpeakerProfile], config: SpeakerMatchingConfig):
        self.profiles = {p.id: p for p in profiles}
        self.config = config

    def find_matches(self, embedding: np.ndarray) -> List[SpeakerMatch]:
        """
        Find matching speakers for an unknown embedding.

        Returns matches sorted by similarity (highest first).
        Each match includes confidence level:
        - "high": similarity >= 0.75, can auto-assign
        - "medium": similarity 0.60-0.75, suggest to user
        - "low": similarity < 0.60, likely new speaker
        """
        matches = []

        # Normalize input embedding
        embedding = embedding / (np.linalg.norm(embedding) + 1e-8)

        for profile in self.profiles.values():
            if profile.centroid is None:
                continue

            # Skip profiles with insufficient samples
            if profile.sample_count < self.config.MIN_SEGMENTS_FOR_PROFILE:
                continue

            # Compute cosine similarity
            similarity = float(np.dot(embedding, profile.centroid))

            # Determine confidence level
            if similarity >= self.config.AUTO_ASSIGN_THRESHOLD:
                confidence = "high"
            elif similarity >= self.config.SUGGESTION_THRESHOLD:
                confidence = "medium"
            else:
                confidence = "low"

            # Adjust similarity by profile confidence
            # (profiles with more samples are more reliable)
            adjusted_similarity = similarity * (0.8 + 0.2 * profile.confidence)

            matches.append(SpeakerMatch(
                speaker_id=profile.id,
                speaker_name=profile.name,
                raw_similarity=similarity,
                adjusted_similarity=adjusted_similarity,
                confidence=confidence,
                profile_sample_count=profile.sample_count
            ))

        # Sort by adjusted similarity descending
        matches.sort(key=lambda x: x.adjusted_similarity, reverse=True)
        return matches

    def auto_identify(self, embedding: np.ndarray) -> Optional[SpeakerMatch]:
        """
        Attempt automatic speaker identification.

        Returns match only if confidence is high enough for auto-assignment.
        """
        matches = self.find_matches(embedding)

        if matches and matches[0].confidence == "high":
            return matches[0]

        return None
```

### 6.3 Profile Drift Handling

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    SPEAKER PROFILE DRIFT HANDLING                                │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  PROBLEM: Voices change over time due to:                                       │
│  • Health (cold, fatigue)                                                       │
│  • Recording conditions (different microphones, rooms)                          │
│  • Natural aging                                                                │
│                                                                                  │
│  SOLUTION: Adaptive Centroid with Slow Learning Rate                            │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                          │   │
│  │  On each new verified sample:                                            │   │
│  │                                                                          │   │
│  │    centroid_new = α × centroid_old + (1 - α) × new_embedding            │   │
│  │                                                                          │   │
│  │  where α = 0.95 (adaptation rate)                                        │   │
│  │                                                                          │   │
│  │  This means:                                                             │   │
│  │  • New sample contributes only 5% to centroid                           │   │
│  │  • 50% change requires ~14 new samples                                  │   │
│  │  • Prevents sudden profile corruption from outliers                     │   │
│  │  • Allows gradual adaptation to voice changes                           │   │
│  │                                                                          │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  SAMPLE DIVERSITY MAINTENANCE:                                                  │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  • Keep maximum 50 samples per speaker                                   │   │
│  │  • When limit reached, prune most similar samples                       │   │
│  │  • This maintains sample diversity across:                              │   │
│  │    - Different recording sessions                                       │   │
│  │    - Different speaking styles (calm, excited)                          │   │
│  │    - Different audio quality conditions                                 │   │
│  │                                                                          │   │
│  │  Pruning algorithm:                                                      │   │
│  │  1. Compute pairwise similarities between all samples                   │   │
│  │  2. Find sample with highest average similarity to others               │   │
│  │  3. Remove that sample (most redundant)                                 │   │
│  │  4. Repeat until under limit                                            │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  PROFILE STALENESS DETECTION:                                                   │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Track last_seen_at for each speaker profile                            │   │
│  │                                                                          │   │
│  │  If not seen in 90 days:                                                 │   │
│  │  • Lower auto-assign threshold to 0.80 (require higher confidence)      │   │
│  │  • Show warning: "Haven't seen [Name] in 3 months - confirm match?"     │   │
│  │                                                                          │   │
│  │  If not seen in 180 days:                                                │   │
│  │  • Disable auto-assign entirely for this speaker                        │   │
│  │  • Always require manual confirmation                                   │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 7. Audio Quality & Constraints

### 7.1 Audio Quality Requirements

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                         AUDIO QUALITY REQUIREMENTS                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  MINIMUM REQUIREMENTS (Processing will fail below these):                        │
│  ─────────────────────────────────────────────────────────                      │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Parameter              Minimum          Recommended                     │   │
│  │  ─────────              ───────          ───────────                     │   │
│  │  Sample Rate            8,000 Hz         16,000 Hz                       │   │
│  │  Bit Depth              16-bit           16-bit                          │   │
│  │  Channels               1 (mono)         1 (mono)                        │   │
│  │  Duration               5 seconds        30+ seconds                     │   │
│  │  Max Duration           4 hours          1-2 hours                       │   │
│  │  File Size              -                < 2 GB                          │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  SIGNAL-TO-NOISE RATIO (SNR) IMPACT:                                           │
│  ────────────────────────────────────                                           │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  SNR Level      Quality          Impact on Processing                   │   │
│  │  ─────────      ───────          ─────────────────────                   │   │
│  │  > 30 dB        Excellent        Full accuracy, reliable speaker ID     │   │
│  │  20-30 dB       Good             Minor accuracy loss, reliable speaker  │   │
│  │  10-20 dB       Acceptable       Noticeable WER increase, speaker ID ok │   │
│  │  < 10 dB        Poor             Significant errors, speaker ID unreliab│   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  AUDIO VALIDATION ON UPLOAD:                                                    │
│  ────────────────────────────                                                   │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  1. Check file format (MP3, WAV, M4A, MP4, WebM, OGG, FLAC)             │   │
│  │  2. Extract audio metadata (sample rate, channels, duration)            │   │
│  │  3. Estimate SNR from first 30 seconds                                  │   │
│  │  4. Detect if audio is mostly silence                                   │   │
│  │                                                                          │   │
│  │  Error messages:                                                         │   │
│  │  • "Unsupported format. Please use MP3, WAV, M4A, or MP4."              │   │
│  │  • "Audio too short. Minimum 5 seconds required."                       │   │
│  │  • "Audio too long. Maximum 4 hours supported."                         │   │
│  │  • "Audio quality too low. Please use a better recording."              │   │
│  │                                                                          │   │
│  │  Warnings (processing continues):                                        │   │
│  │  • "Low audio quality detected. Transcription may be less accurate."    │   │
│  │  • "Audio is mostly silence. Only speech portions will be processed."   │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 7.2 Segment Duration Requirements for Speaker ID

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                    SEGMENT DURATION FOR SPEAKER ID                               │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ECAPA-TDNN embedding quality depends heavily on audio duration:                │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Duration        Embedding Quality     Use Case                         │   │
│  │  ────────        ─────────────────     ────────                         │   │
│  │  < 1 second      Very Poor             Not reliable, skip               │   │
│  │  1-2 seconds     Poor                  Unreliable, use only if no other │   │
│  │  2-3 seconds     Acceptable            Can use but prefer longer        │   │
│  │  3-5 seconds     Good                  Reliable for most cases          │   │
│  │  5-10 seconds    Excellent             Ideal for profile building       │   │
│  │  > 10 seconds    Excellent             Diminishing returns above 10s    │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  SEGMENT SELECTION STRATEGY:                                                    │
│  ───────────────────────────                                                    │
│  For embedding extraction, prioritize segments by:                              │
│                                                                                  │
│  1. Duration (prefer 5-10 seconds)                                              │
│  2. Audio quality (SNR estimate)                                                │
│  3. No speaker overlap                                                          │
│  4. No music/noise in background                                                │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Quality Score = duration_score × snr_score × overlap_penalty           │   │
│  │                                                                          │   │
│  │  where:                                                                  │   │
│  │    duration_score = min(duration_seconds / 5, 1.0)                      │   │
│  │    snr_score = min(snr_db / 30, 1.0)                                    │   │
│  │    overlap_penalty = 0.5 if overlapping else 1.0                        │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  SHORT SEGMENT HANDLING:                                                        │
│  ───────────────────────                                                        │
│  • Segments < 1 second: Merge with adjacent same-speaker segment               │
│  • Segments 1-3 seconds: Use for transcription, avoid for embedding            │
│  • If only short segments available: Concatenate multiple segments              │
│    from same speaker (up to 10 seconds) for embedding extraction               │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 8. Edge Cases & Error Handling

### 8.1 Overlapping Speakers

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        OVERLAPPING SPEAKERS HANDLING                             │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  DETECTION:                                                                      │
│  ──────────                                                                      │
│  • pyannote.audio can detect overlapping speech regions                         │
│  • Mark segments with overlap flag                                              │
│  • Identify speaker changes < 500ms as potential overlap                        │
│                                                                                  │
│  HANDLING STRATEGIES:                                                           │
│  ────────────────────                                                           │
│                                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Scenario                          Action                                │   │
│  │  ────────                          ──────                                │   │
│  │  Full overlap (both speaking)      Mark segment as "Multiple Speakers"  │   │
│  │                                    Don't use for embedding extraction   │   │
│  │                                    Include in transcript with label     │   │
│  │                                                                          │   │
│  │  Partial overlap (interruption)    Split at overlap boundary            │   │
│  │                                    Assign primary speaker to each part  │   │
│  │                                    Mark overlap region                  │   │
│  │                                                                          │   │
│  │  Quick interjection (<1s)          Keep with primary speaker            │   │
│  │                                    Note interjection in metadata        │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  UI DISPLAY:                                                                     │
│  ───────────                                                                     │
│  • Show overlap indicator on segments with speaker overlap                      │
│  • Allow user to manually split/merge overlapping segments                      │
│  • Don't count overlapping segments toward speaker profile                      │
│                                                                                  │
│  EMBEDDING EXTRACTION:                                                          │
│  ─────────────────────                                                          │
│  • NEVER extract embeddings from overlapping segments                           │
│  • Reduces speaker profile contamination                                        │
│  • Prefer clean, single-speaker segments for profile building                   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 8.2 Background Noise Handling

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                        BACKGROUND NOISE HANDLING                                 │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  NOISE TYPES AND IMPACT:                                                        │
│  ───────────────────────                                                        │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Noise Type           Impact on Transcription   Impact on Speaker ID   │   │
│  │  ──────────           ──────────────────────   ─────────────────────   │   │
│  │  White noise          Low (Whisper handles)    Low                     │   │
│  │  Background music     Medium                   High (can corrupt)      │   │
│  │  Other voices         High                     Very High               │   │
│  │  Keyboard/clicks      Low                      Low                     │   │
│  │  Echo/reverb          Medium                   Medium                  │   │
│  │  Wind/handling        Medium                   Medium                  │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  PREPROCESSING OPTIONS (User-configurable):                                     │
│  ──────────────────────────────────────────                                     │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Option                   Default    Description                        │   │
│  │  ──────                   ───────    ───────────                        │   │
│  │  Noise reduction          Off        Apply spectral gating              │   │
│  │  Auto-gain                On         Normalize audio levels             │   │
│  │  High-pass filter         On         Remove rumble < 80Hz               │   │
│  │  Music detection          On         Detect and skip music sections     │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  QUALITY SCORING FOR SEGMENTS:                                                  │
│  ─────────────────────────────                                                  │
│  Each segment gets a quality score (0-1) based on:                              │
│  • SNR estimate                                                                 │
│  • Presence of music/noise                                                      │
│  • Speaker overlap                                                              │
│  • Duration                                                                     │
│                                                                                  │
│  Quality score is used to:                                                      │
│  • Prioritize segments for embedding extraction                                 │
│  • Weight confidence in speaker matching                                        │
│  • Display quality indicator in UI                                              │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 8.3 Low Confidence Matching Workflow

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                      LOW CONFIDENCE MATCHING WORKFLOW                            │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  SCENARIO: System detects a speaker with 65% similarity to "Bill"               │
│  (Between suggestion threshold 60% and auto-assign threshold 75%)               │
│                                                                                  │
│  AUTOMATIC BEHAVIOR:                                                            │
│  ───────────────────                                                            │
│  1. Keep temporary speaker ID ("Speaker C")                                     │
│  2. Store potential match in metadata                                           │
│  3. Show suggestion in UI, don't auto-assign                                    │
│                                                                                  │
│  UI PRESENTATION:                                                               │
│  ────────────────                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Sidebar Speaker Card:                                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────────┐  │   │
│  │  │  ○ Speaker C (unidentified)                     22%              │  │   │
│  │  │    6 segments • 9:58 speaking                                     │  │   │
│  │  │                                                                    │  │   │
│  │  │    ┌──────────────────────────────────────────────────────────┐   │  │   │
│  │  │    │  💡 Possible match:                                      │   │  │   │
│  │  │    │     Bill (65% confidence)                                │   │  │   │
│  │  │    │                                                           │   │  │   │
│  │  │    │     [▶ Play Sample]  [Confirm Bill]  [Not Bill]          │   │  │   │
│  │  │    └──────────────────────────────────────────────────────────┘   │  │   │
│  │  │                                                                    │  │   │
│  │  │    [Label as someone else...]                                     │  │   │
│  │  └───────────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  USER ACTIONS:                                                                  │
│  ─────────────                                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Action              Result                                             │   │
│  │  ──────              ──────                                             │   │
│  │  [Confirm Bill]      - Assign all Speaker C segments to Bill           │   │
│  │                      - Add embeddings to Bill's profile                │   │
│  │                      - Update Bill's last_seen_at                      │   │
│  │                      - Increase Bill's confidence score                │   │
│  │                                                                          │   │
│  │  [Not Bill]          - Keep Speaker C as unidentified                  │   │
│  │                      - Remove Bill from suggestions for this speaker   │   │
│  │                      - Log negative match (don't use for training)     │   │
│  │                                                                          │   │
│  │  [Label as...]       - Open full labeling modal                        │   │
│  │                      - Can select different speaker or create new      │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  LEARNING FROM REJECTIONS:                                                      │
│  ─────────────────────────                                                      │
│  • Store rejected matches to avoid repeating wrong suggestions                  │
│  • If same speaker repeatedly rejected for a profile:                           │
│    - System learns this voice is distinct from that speaker                     │
│    - Increases matching threshold for that specific pair                        │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 8.4 Error Recovery

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           ERROR RECOVERY STRATEGIES                              │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  PROCESSING ERRORS:                                                             │
│  ──────────────────                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  Error                        Recovery                                  │   │
│  │  ─────                        ────────                                  │   │
│  │  Whisper API timeout          Retry with exponential backoff (3x)      │   │
│  │  Whisper API rate limit       Queue and retry after delay              │   │
│  │  pyannote model OOM           Process in smaller chunks                │   │
│  │  Audio extraction fail        Try alternative decoder (FFmpeg fallback)│   │
│  │  Embedding extraction fail    Skip segment, log warning                │   │
│  │  Database write fail          Retry 3x, then mark recording as failed  │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
│  CHECKPOINT SYSTEM:                                                             │
│  ──────────────────                                                             │
│  Save progress at each stage:                                                   │
│  1. Audio extracted → checkpoint                                                │
│  2. VAD complete → checkpoint                                                   │
│  3. Diarization complete → checkpoint                                           │
│  4. Transcription complete → checkpoint                                         │
│  5. Alignment complete → checkpoint                                             │
│  6. Embeddings extracted → checkpoint                                           │
│  7. Matching complete → final                                                   │
│                                                                                  │
│  On resume after crash:                                                         │
│  • Check latest checkpoint                                                      │
│  • Resume from that stage                                                       │
│  • Notify user: "Resuming processing from [stage]..."                          │
│                                                                                  │
│  GRACEFUL DEGRADATION:                                                          │
│  ─────────────────────                                                          │
│  ┌─────────────────────────────────────────────────────────────────────────┐   │
│  │  If diarization fails:                                                  │   │
│  │    → Continue with single-speaker transcript                            │   │
│  │    → Show warning: "Could not identify separate speakers"              │   │
│  │                                                                          │   │
│  │  If speaker matching fails:                                              │   │
│  │    → Keep temporary speaker IDs                                         │   │
│  │    → Allow manual labeling                                              │   │
│  │    → Show warning: "Automatic speaker matching unavailable"             │   │
│  │                                                                          │   │
│  │  If embedding extraction fails:                                          │   │
│  │    → Skip that segment's embedding                                      │   │
│  │    → Still allow manual labeling                                        │   │
│  │    → Profile will have fewer samples                                    │   │
│  └─────────────────────────────────────────────────────────────────────────┘   │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 9. Data Models & Storage

### 9.1 Database Schema

```sql
-- =====================================================
-- CORE TABLES
-- =====================================================

-- Recording sessions (one per upload or capture)
CREATE TABLE recordings (
    id              TEXT PRIMARY KEY,
    title           TEXT NOT NULL,
    description     TEXT,
    source_type     TEXT NOT NULL CHECK (source_type IN ('screen_capture', 'audio_file', 'live_mic')),
    source_path     TEXT,
    duration_ms     INTEGER,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    processing_stage TEXT,  -- Current stage if processing
    audio_format    TEXT,
    sample_rate     INTEGER DEFAULT 16000,
    snr_estimate_db REAL,   -- Estimated signal-to-noise ratio
    error_message   TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata        TEXT    -- JSON blob for extensibility
);

CREATE INDEX idx_recordings_status ON recordings(status);
CREATE INDEX idx_recordings_created ON recordings(created_at DESC);

-- Speaker profiles (learned from user labels)
CREATE TABLE speakers (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    email           TEXT,
    avatar_path     TEXT,
    color           TEXT,                          -- Hex color for UI
    embedding_centroid BLOB,                       -- 192-dim float32 vector
    sample_count    INTEGER DEFAULT 0,
    confidence      REAL DEFAULT 0.0 CHECK (confidence >= 0 AND confidence <= 1),
    is_active       BOOLEAN DEFAULT TRUE,          -- FALSE for deleted/merged
    first_seen_at   DATETIME,
    last_seen_at    DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata        TEXT
);

CREATE INDEX idx_speakers_name ON speakers(name);
CREATE INDEX idx_speakers_active ON speakers(is_active);

-- Voice embedding samples for each speaker
CREATE TABLE speaker_embeddings (
    id              TEXT PRIMARY KEY,
    speaker_id      TEXT NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    segment_id      TEXT REFERENCES segments(id) ON DELETE SET NULL,
    embedding       BLOB NOT NULL,                 -- 192-dim float32 vector
    quality_score   REAL DEFAULT 1.0 CHECK (quality_score >= 0 AND quality_score <= 1),
    duration_ms     INTEGER,                       -- Source segment duration
    is_from_overlap BOOLEAN DEFAULT FALSE,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_speaker_embeddings_speaker ON speaker_embeddings(speaker_id);
CREATE INDEX idx_speaker_embeddings_quality ON speaker_embeddings(quality_score DESC);

-- Transcript segments (individual utterances)
CREATE TABLE segments (
    id              TEXT PRIMARY KEY,
    recording_id    TEXT NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    speaker_id      TEXT REFERENCES speakers(id) ON DELETE SET NULL,
    temp_speaker_id TEXT,                          -- e.g., "Speaker A" before labeling
    text            TEXT NOT NULL,
    start_ms        INTEGER NOT NULL,
    end_ms          INTEGER NOT NULL,

    -- Confidence scores
    transcription_confidence REAL,                 -- From Whisper
    speaker_confidence       REAL,                 -- From matching algorithm

    -- Quality indicators
    quality_score   REAL,                          -- Overall quality (0-1)
    has_overlap     BOOLEAN DEFAULT FALSE,
    snr_estimate_db REAL,

    -- User actions
    is_user_verified BOOLEAN DEFAULT FALSE,        -- User confirmed speaker
    is_text_edited   BOOLEAN DEFAULT FALSE,        -- User edited text

    -- Voice embedding for this segment
    embedding       BLOB,                          -- 192-dim float32 (if extracted)

    -- Word-level timestamps (JSON array)
    word_timestamps TEXT,

    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,

    CHECK (end_ms > start_ms)
);

CREATE INDEX idx_segments_recording ON segments(recording_id);
CREATE INDEX idx_segments_speaker ON segments(speaker_id);
CREATE INDEX idx_segments_time ON segments(recording_id, start_ms);
CREATE INDEX idx_segments_temp_speaker ON segments(recording_id, temp_speaker_id);

-- Full-text search on transcript text
CREATE VIRTUAL TABLE segments_fts USING fts5(
    text,
    content='segments',
    content_rowid='rowid',
    tokenize='porter unicode61'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER segments_ai AFTER INSERT ON segments BEGIN
    INSERT INTO segments_fts(rowid, text) VALUES (new.rowid, new.text);
END;

CREATE TRIGGER segments_ad AFTER DELETE ON segments BEGIN
    INSERT INTO segments_fts(segments_fts, rowid, text) VALUES('delete', old.rowid, old.text);
END;

CREATE TRIGGER segments_au AFTER UPDATE ON segments BEGIN
    INSERT INTO segments_fts(segments_fts, rowid, text) VALUES('delete', old.rowid, old.text);
    INSERT INTO segments_fts(rowid, text) VALUES (new.rowid, new.text);
END;

-- =====================================================
-- MATCHING & SUGGESTIONS
-- =====================================================

-- Track suggested matches for user confirmation
CREATE TABLE speaker_suggestions (
    id              TEXT PRIMARY KEY,
    recording_id    TEXT NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    temp_speaker_id TEXT NOT NULL,
    suggested_speaker_id TEXT NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    similarity      REAL NOT NULL,
    status          TEXT DEFAULT 'pending' CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at     DATETIME
);

CREATE INDEX idx_suggestions_recording ON speaker_suggestions(recording_id, status);

-- Track rejected matches to avoid repeating bad suggestions
CREATE TABLE rejected_matches (
    id              TEXT PRIMARY KEY,
    speaker_id      TEXT NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    embedding       BLOB NOT NULL,                 -- The rejected embedding
    similarity      REAL NOT NULL,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_rejected_speaker ON rejected_matches(speaker_id);

-- =====================================================
-- SETTINGS
-- =====================================================

CREATE TABLE settings (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO settings (key, value) VALUES
    ('transcription_model', 'whisper-1'),
    ('transcription_language', 'auto'),
    ('use_local_whisper', 'false'),
    ('diarization_enabled', 'true'),
    ('speaker_matching_enabled', 'true'),
    ('auto_assign_threshold', '0.75'),
    ('suggestion_threshold', '0.60'),
    ('min_segments_for_profile', '3'),
    ('min_segment_duration_ms', '3000');

-- =====================================================
-- EXPORT HISTORY
-- =====================================================

CREATE TABLE exports (
    id              TEXT PRIMARY KEY,
    recording_id    TEXT NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    format          TEXT NOT NULL CHECK (format IN ('txt', 'srt', 'vtt', 'json', 'docx')),
    file_path       TEXT,
    options         TEXT,  -- JSON of export options used
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_exports_recording ON exports(recording_id);
```

### 9.2 File System Structure

```
~/Library/Application Support/Scribe/
├── data/
│   └── scribe.db              # SQLite database (all structured data)
│
├── uploads/
│   ├── {recording_id}.wav     # Converted audio (original preserved)
│   └── {recording_id}_orig.*  # Original uploaded file
│
├── models/
│   ├── ecapa-tdnn/            # SpeechBrain speaker embedding model
│   │   ├── embedding_model.ckpt
│   │   └── config.json
│   │
│   ├── whisper/               # Local Whisper model (if enabled)
│   │   └── large-v3/
│   │       ├── model.bin
│   │       └── config.json
│   │
│   └── pyannote/              # Speaker diarization model
│       └── speaker-diarization-3.1/
│
├── exports/
│   └── {export_id}.*          # Exported transcript files
│
├── cache/
│   ├── checkpoints/           # Processing checkpoints for resume
│   └── temp/                  # Temporary processing files
│
└── logs/
    ├── app.log                # Application logs
    └── processing.log         # Detailed processing logs
```

---

## 10. API Specification

### 10.1 REST API Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| **Recordings** | | |
| GET | `/api/v1/recordings` | List all recordings |
| POST | `/api/v1/recordings` | Upload new recording |
| GET | `/api/v1/recordings/{id}` | Get recording with segments |
| PATCH | `/api/v1/recordings/{id}` | Update recording metadata |
| DELETE | `/api/v1/recordings/{id}` | Delete recording |
| POST | `/api/v1/recordings/{id}/export` | Export transcript |
| POST | `/api/v1/recordings/{id}/reprocess` | Reprocess recording |
| **Speakers** | | |
| GET | `/api/v1/speakers` | List all speakers |
| POST | `/api/v1/speakers` | Create speaker |
| GET | `/api/v1/speakers/{id}` | Get speaker details |
| PATCH | `/api/v1/speakers/{id}` | Update speaker |
| DELETE | `/api/v1/speakers/{id}` | Delete speaker |
| POST | `/api/v1/speakers/{id}/merge` | Merge speakers |
| **Segments** | | |
| PATCH | `/api/v1/segments/{id}` | Update segment |
| POST | `/api/v1/segments/bulk-label` | Label multiple segments |
| GET | `/api/v1/segments/{id}/suggestions` | Get speaker suggestions |
| **Search** | | |
| GET | `/api/v1/search` | Full-text search |
| **Capture** | | |
| POST | `/api/v1/capture/start` | Start screen capture |
| POST | `/api/v1/capture/stop` | Stop capture |
| GET | `/api/v1/capture/status` | Get capture status |

### 10.2 WebSocket Events

| Event | Direction | Description |
|-------|-----------|-------------|
| `processing:started` | Server → Client | Recording processing started |
| `processing:progress` | Server → Client | Progress update (stage, percentage) |
| `processing:segment` | Server → Client | New segment transcribed |
| `processing:speaker_detected` | Server → Client | New speaker detected with suggestions |
| `processing:completed` | Server → Client | Processing finished |
| `processing:failed` | Server → Client | Processing failed with error |
| `capture:audio_level` | Server → Client | Real-time audio level (for live capture) |

---

## 11. User Interface Specification

### 11.1 Key Screens

1. **Dashboard** - Recording list, upload, speaker overview
2. **Transcript View** - Segments, playback, speaker sidebar
3. **Speaker Labeling Modal** - Suggestions, create new, apply
4. **Speaker Management** - Profile list, edit, merge, delete
5. **Settings** - Transcription options, thresholds, API keys

### 11.2 Speaker Labeling Modal (Detailed)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│  Label Speaker                                                              [X] │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  Who is "Speaker C"?                                                            │
│                                                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  🔊 "I've been working on the frontend components, mainly focusing on..." │  │
│  │                                                                            │  │
│  │     [▶ Play Sample]     Duration: 8.3s    Quality: ★★★★☆                  │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  📊 Speaker Statistics                                                    │  │
│  │     • 6 segments in this recording                                        │  │
│  │     • 9:58 total speaking time                                            │  │
│  │     • Quality: Good (avg. SNR 24dB)                                       │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  ───────────────────────────────────────────────────────────────────────────────│
│                                                                                  │
│  Suggested Matches:                                                             │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  ◉ Alex Chen                                         78% match            │  │
│  │    🟢 High confidence - can auto-assign                                   │  │
│  │    Last seen: 3 days ago • 45 recordings • 312 samples                   │  │
│  │                                                        [Confirm Alex]     │  │
│  ├───────────────────────────────────────────────────────────────────────────┤  │
│  │  ○ Jordan Smith                                      62% match            │  │
│  │    🟡 Medium confidence - verify before assigning                         │  │
│  │    Last seen: 1 week ago • 12 recordings • 48 samples                    │  │
│  │                                                        [Select]           │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  ─────────────────────────── OR CREATE NEW ──────────────────────────────────── │
│                                                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  ○ Create new speaker profile                                             │  │
│  │                                                                            │  │
│  │    Name: [________________________]                                        │  │
│  │    Email: [______________________] (optional)                              │  │
│  │                                                                            │  │
│  │    ⚠️  You've labeled 6 segments (9:58 audio). This meets the minimum    │  │
│  │       requirement of 3 segments for a reliable profile.                   │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Options:                                                                  │  │
│  │  ☑ Apply to all 6 segments from Speaker C in this recording              │  │
│  │  ☑ Remember this speaker for future recordings                           │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│                                            [Cancel]  [Apply Label]              │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 12. Performance Benchmarks

### 12.1 Target Performance Metrics

| Metric | Target | Measurement Method |
|--------|--------|-------------------|
| **Transcription Speed** | < 0.3x real-time | 1hr audio in < 20 min |
| **Diarization Speed** | < 0.5x real-time | 1hr audio in < 30 min |
| **Total Processing** | < 0.6x real-time | 1hr audio in < 36 min |
| **Speaker Matching** | < 100ms per segment | Time from embedding to match |
| **Search Latency** | < 200ms | Query to results displayed |
| **UI Response** | < 100ms | All user interactions |
| **App Launch** | < 2s | Cold start to ready |
| **Memory Usage** | < 500MB baseline | During idle |
| **Memory Peak** | < 2GB | During ML inference |

### 12.2 Accuracy Targets

| Metric | Target | Conditions |
|--------|--------|------------|
| **Transcription WER** | < 10% | Clean English audio, SNR > 20dB |
| **Transcription WER** | < 15% | Moderate noise, SNR 10-20dB |
| **Diarization DER** | < 15% | 2-4 speakers, minimal overlap |
| **Diarization DER** | < 25% | 5+ speakers or significant overlap |
| **Speaker ID Accuracy** | > 95% | After 10+ labeled samples |
| **Speaker ID Accuracy** | > 85% | After 5 labeled samples |
| **Speaker ID Accuracy** | > 70% | After 3 labeled samples (minimum) |

### 12.3 Resource Constraints

| Resource | Limit | Notes |
|----------|-------|-------|
| **Disk Space per Recording** | ~50MB/hour | WAV audio + embeddings + metadata |
| **Database Size** | < 1GB for 100 hours | With compression, without audio |
| **Model Storage** | ~1.5GB | All ML models combined |
| **GPU Memory (Apple Silicon)** | < 4GB | For local Whisper inference |
| **Concurrent Recordings** | 1 | Sequential processing by default |

---

## 13. Security & Privacy

### 13.1 Data Protection

| Data Type | Storage | Encryption | Retention |
|-----------|---------|------------|-----------|
| Audio files | Local only | None (optional SQLCipher) | Until user deletes |
| Transcripts | SQLite | None (optional SQLCipher) | Until user deletes |
| Voice embeddings | SQLite BLOB | None | With speaker profile |
| API keys | macOS Keychain | System encryption | Until removed |

### 13.2 Privacy Principles

1. **Local-First**: All processing on-device by default
2. **No Telemetry**: No usage data sent without consent
3. **No Cloud Storage**: Audio never uploaded unless user explicitly chooses
4. **Non-Reversible Embeddings**: Voice embeddings cannot reconstruct audio
5. **User Control**: All data can be deleted; no data lock-in

### 13.3 macOS Permissions Required

| Permission | When Requested | Purpose |
|------------|----------------|---------|
| Screen Recording | First screen capture | Access system audio via ScreenCaptureKit |
| Microphone | First mic recording | Record from microphone |
| Files & Folders | Never (uses open panels) | User-selected files only |

---

## 14. Development Phases

### Phase 1: Core MVP (4-6 weeks)

**Goal**: Basic transcription from audio files with manual speaker labeling

**Deliverables**:
- [ ] FastAPI backend with processing pipeline
- [ ] Whisper integration (API)
- [ ] pyannote diarization
- [ ] SQLite database schema
- [ ] Next.js frontend with transcript viewer
- [ ] File upload (MP3, WAV, M4A, MP4)
- [ ] Speaker labeling UI
- [ ] Export (TXT, SRT)

**Acceptance Test**: Upload 1-hour meeting recording, see transcript with Speaker A/B/C labels, manually label speakers, export to SRT.

### Phase 2: Speaker Learning (3-4 weeks)

**Goal**: Learn speaker identities from labels, auto-identify in new recordings

**Deliverables**:
- [ ] ECAPA-TDNN embedding extraction
- [ ] Speaker profile management
- [ ] Cosine similarity matching
- [ ] Auto-assignment (>0.75 confidence)
- [ ] Suggestions UI (0.60-0.75 confidence)
- [ ] Profile drift handling
- [ ] Minimum sample warnings

**Acceptance Test**: Label "Bill" in recording 1, upload recording 2 with Bill, system auto-identifies Bill with 80%+ confidence.

### Phase 3: Mac Native Integration (4-5 weeks)

**Goal**: Native macOS app with screen/audio capture

**Deliverables**:
- [ ] Tauri 2.0 app wrapper
- [ ] ScreenCaptureKit integration
- [ ] System audio capture
- [ ] Microphone capture
- [ ] Permission handling
- [ ] Menu bar controls
- [ ] Local Whisper option (faster-whisper)

**Acceptance Test**: Start screen recording during Zoom call, stop recording, see transcript with identified speakers.

### Phase 4: Polish & Advanced (3-4 weeks)

**Goal**: Production-ready with advanced features

**Deliverables**:
- [ ] Full-text search
- [ ] Export to DOCX
- [ ] Keyboard shortcuts
- [ ] Batch processing
- [ ] Performance optimization
- [ ] Error recovery & checkpoints
- [ ] Settings UI

**Acceptance Test**: Search across 50+ recordings in <200ms, batch process 10 files overnight with resume on failure.

---

## 15. Acceptance Criteria

### 15.1 Core Functionality Acceptance

| ID | Test Case | Expected Result | Priority |
|----|-----------|-----------------|----------|
| AC-1 | Upload 1hr MP4 meeting recording | Processing completes in <25 min | P0 |
| AC-2 | View transcript of 4-person meeting | 4 distinct speakers identified | P0 |
| AC-3 | Play audio segment | Audio plays synced with text highlight | P0 |
| AC-4 | Label Speaker A as "Bill" | All Speaker A segments update to Bill | P0 |
| AC-5 | Upload new recording with Bill | Bill auto-identified (>75% match) | P0 |
| AC-6 | Export to SRT | Valid SRT file with speaker names | P0 |
| AC-7 | Search "quarterly report" | Matching segments found in <200ms | P1 |
| AC-8 | Delete recording | Recording and segments removed | P0 |
| AC-9 | Merge two speakers | Segments reassigned, profiles combined | P1 |
| AC-10 | Process low-quality audio | Warning shown, processing continues | P1 |

### 15.2 Speaker Learning Acceptance

| ID | Test Case | Expected Result | Priority |
|----|-----------|-----------------|----------|
| SL-1 | Label 3 segments for new speaker | Profile created with warning about minimum samples | P0 |
| SL-2 | Label 5 segments for speaker | Profile created, confidence ~0.25 | P0 |
| SL-3 | Confirm suggested match | Speaker assigned, profile updated | P0 |
| SL-4 | Reject suggested match | Suggestion dismissed, not repeated | P0 |
| SL-5 | Speaker not seen for 90 days | Higher threshold for auto-assign | P1 |
| SL-6 | Same speaker, different mic | Still recognized with >60% similarity | P1 |
| SL-7 | Label segment with overlap | Warning shown, embedding not extracted | P1 |

### 15.3 Error Handling Acceptance

| ID | Test Case | Expected Result | Priority |
|----|-----------|-----------------|----------|
| EH-1 | Upload corrupted audio file | Error message: "File could not be processed" | P0 |
| EH-2 | API timeout during transcription | Retry automatically, succeed on retry | P0 |
| EH-3 | App crash during processing | Resume from checkpoint on restart | P1 |
| EH-4 | Disk full during processing | Graceful error, partial data saved | P1 |
| EH-5 | No speakers detected | Show single "Speaker" for all segments | P1 |

---

## Appendix A: Glossary

| Term | Definition |
|------|------------|
| **Diarization** | Segmenting audio by speaker ("who spoke when") |
| **WER** | Word Error Rate - transcription accuracy metric (lower is better) |
| **DER** | Diarization Error Rate - speaker segmentation accuracy (lower is better) |
| **EER** | Equal Error Rate - speaker verification accuracy at equal FAR/FRR |
| **Embedding** | Fixed-size vector representing voice characteristics (192-dim for ECAPA-TDNN) |
| **Centroid** | Average embedding representing a speaker's typical voice |
| **SNR** | Signal-to-Noise Ratio - measure of audio quality (higher is better) |
| **VAD** | Voice Activity Detection - finding speech vs silence |
| **Real-time factor** | Processing time / audio duration (0.3x = 1hr processed in 18min) |

---

## Appendix B: References

1. [pyannote.audio](https://github.com/pyannote/pyannote-audio) - Speaker diarization
2. [faster-whisper](https://github.com/guillaumekln/faster-whisper) - Optimized Whisper
3. [SpeechBrain](https://github.com/speechbrain/speechbrain) - Voice embeddings (ECAPA-TDNN)
4. [Tauri](https://tauri.app/) - Desktop app framework
5. [ScreenCaptureKit](https://developer.apple.com/documentation/screencapturekit) - macOS screen capture
6. Whisper Paper: "Robust Speech Recognition via Large-Scale Weak Supervision" (Radford et al., 2022)
7. ECAPA-TDNN Paper: "Emphasized Channel Attention for Speaker Verification" (Desplanques et al., 2020)

---

*End of Specification v2.0*
