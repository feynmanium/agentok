# Technical Specification: Scribe (macOS)

**Version:** 1.1 (Enhanced)
**Date:** January 25, 2026
**Status:** Approved for Implementation

---

## 1. Executive Summary

Scribe is a local-first macOS application designed to record screen activity and audio, transcribe speech, and perform speaker diarization. Its core differentiator is an **Active Learning System**: as users manually label speaker segments (e.g., "Speaker A" is "Bill"), the system generates voice embeddings to automatically identify that speaker in future recordings.

**Core Philosophy:** Privacy-first (local processing), user-in-the-loop learning, and degradation-graceful (it admits when it doesn't know a speaker).

---

## 2. System Architecture

### 2.1 High-Level Architecture Diagram

The system utilizes a **Sidecar Architecture**:

1. **Presentation Layer (Tauri/Next.js):** Handles UI, OS-level window management, and screen capture rendering.
2. **Logic Layer (Python Sidecar):** A compiled FastAPI binary managed by Tauri. Handles heavy ML lifting (PyTorch, Whisper, Pyannote).
3. **Data Layer (SQLite):** Stores metadata, transcripts, and high-dimensional vector embeddings.

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              SCRIBE APPLICATION                                  │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                    TAURI 2.0 SHELL (Rust)                                  │ │
│  │  ┌──────────────────┐  ┌──────────────────┐  ┌──────────────────────────┐  │ │
│  │  │ ScreenCaptureKit │  │   System Tray    │  │   Sidecar Manager        │  │ │
│  │  │   + AVFoundation │  │    Controls      │  │   (spawns Python bin)    │  │ │
│  │  └──────────────────┘  └──────────────────┘  └──────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                          │
│                                       │ WebView                                  │
│                                       ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                    NEXT.JS 15 FRONTEND                                      │ │
│  │  ┌────────────────┐  ┌────────────────┐  ┌──────────────────────────────┐  │ │
│  │  │  Recording UI  │  │ Transcript View│  │   Speaker Management        │  │ │
│  │  │  • Capture     │  │ • Virtualized  │  │   • Label Modal             │  │ │
│  │  │  • Import      │  │ • Playback     │  │   • Merge Speakers          │  │ │
│  │  └────────────────┘  └────────────────┘  └──────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                          │
│                                       │ REST API (localhost:5005)                │
│                                       │ + Shared Secret Token                    │
│                                       ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                    PYTHON SIDECAR (Frozen Binary)                           │ │
│  │                                                                              │ │
│  │  ┌─────────────────────────────────────────────────────────────────────┐   │ │
│  │  │                      PROCESSING PIPELINE                             │   │ │
│  │  │                                                                      │   │ │
│  │  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────────────┐ │   │ │
│  │  │  │ Preproc  │─▶│ Diarize  │─▶│Transcribe│─▶│    Align + Match     │ │   │ │
│  │  │  │Band-pass │  │ Pyannote │  │ Whisper  │  │    Embeddings        │ │   │ │
│  │  │  │300-3400Hz│  │   3.1    │  │ Large-v3 │  │    ECAPA-TDNN        │ │   │ │
│  │  │  └──────────┘  └──────────┘  └──────────┘  └──────────────────────┘ │   │ │
│  │  └─────────────────────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                       │                                          │
│                                       ▼                                          │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │                    SQLITE + sqlite-vss                                      │ │
│  │  ┌──────────────┐  ┌──────────────────────────┐  ┌──────────────────────┐  │ │
│  │  │  recordings  │  │   speaker_embeddings     │  │      segments        │  │ │
│  │  │  speakers    │  │   (VSS virtual table)    │  │      (FTS5)          │  │ │
│  │  └──────────────┘  └──────────────────────────┘  └──────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐ │
│  │  ~/Library/Application Support/Scribe/                                      │ │
│  │  ├── scribe.db          (SQLite database)                                   │ │
│  │  ├── audio/             (AAC/M4A recordings)                                │ │
│  │  └── models/            (ML models)                                         │ │
│  └────────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Stack

| Component | Technology | Justification |
| --- | --- | --- |
| **App Shell** | **Tauri 2.0 (Rust)** | Native macOS performance, smaller bundle than Electron, handles `ScreenCaptureKit` effectively. |
| **Frontend** | **Next.js 15 + React 19** | Modern state management, virtualized lists for long transcripts, static export capability. |
| **Backend** | **FastAPI (Python 3.11)** | Required for the rich ecosystem of Audio ML libraries (pyannote, faster-whisper). |
| **Database** | **SQLite + `sqlite-vss`** | Local single-file DB. `sqlite-vss` enables vector similarity search for speaker matching without an external vector DB. |
| **Transcription** | **Whisper (Large-v3)** | State-of-the-art accuracy. Configurable to use OpenAI API (Cloud) or `faster-whisper` (Local). |
| **Diarization** | **pyannote.audio 3.1** | Best-in-class open-source speaker segmentation. |
| **Embedding** | **ECAPA-TDNN** | Optimized for speaker verification; 192-dimensional embeddings are efficient for storage and comparison. |

---

## 3. Core Features & Acceptance Criteria

### 3.1 Recording & Capture

* **Spec:** Utilize `ScreenCaptureKit` (SCK) for high-performance screen/audio capture and `AVCaptureSession` for microphone input.
* **Devil's Advocate / Edge Cases:**
  * *Permissions:* App must detect missing permissions and guide the user to System Settings > Privacy.
  * *Drift:* Audio/Video desync must be < 100ms.

* **Acceptance Criteria:**
  - [ ] User can select specific window or full screen.
  - [ ] System audio and Mic audio are recorded on separate tracks (or intelligently mixed).
  - [ ] Recording continues even if the target window is minimized.

### 3.2 Transcription & Diarization Pipeline

* **Spec:** Audio is processed in chunks. First Diarization (Who spoke when), then Transcription (What was said), then Alignment.
* **Constraint:** Minimum audio quality checks required.

* **Acceptance Criteria:**
  - [ ] Word Error Rate (WER) < 10% on clear audio.
  - [ ] Diarization Error Rate (DER) < 15% for non-overlapping speech.
  - [ ] **Edge Case:** Overlapping speech > 500ms must be flagged or assigned to "Multiple Speakers" rather than forcing a wrong single-speaker label.

### 3.3 Speaker Learning (The Core Mechanic)

* **Spec:**
  1. Extract 192-dim embedding using ECAPA-TDNN from labeled segments.
  2. Store "Speaker Profile" = Centroid of approved embeddings.
  3. Match new segments using Cosine Similarity.

* **Thresholds (Strict):**
  * **> 0.75:** Auto-Label (High Confidence).
  * **0.60 - 0.75:** Suggest Label (Medium Confidence - UI shows "Is this Bill?").
  * **< 0.60:** Unknown Speaker.

* **Constraints:**
  * **Min Duration:** Segments < 1.5s are ignored for profile creation (insufficient data).
  * **Min Samples:** A speaker profile is only "Active" after 3 manual labels.

* **Acceptance Criteria:**
  - [ ] System does *not* auto-label a speaker with < 3 confirmed samples.
  - [ ] UI visually distinguishes between "Auto-labeled" (dotted line) and "Manually labeled" (solid).

---

## 4. Data Flow & Processing Pipeline

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                           PROCESSING PIPELINE                                    │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                  │
│  1. INGESTION                                                                    │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Input: input.wav (Screen + Mic mix)                                      │  │
│  │  Preprocessing: Band-pass filter (300Hz - 3400Hz) to remove hum/hiss      │  │
│  │  Optional: DeepFilterNet if SNR < 15dB                                    │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                       │                                          │
│                                       ▼                                          │
│  2. SEGMENTATION (Pyannote)                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Output: List[Segment(start, end, speaker_id_cluster)]                    │  │
│  │                                                                            │  │
│  │  Edge Cases:                                                               │  │
│  │  • Overlapping speech > 500ms → "Multiple Speakers"                       │  │
│  │  • Segments < 1.5s → Skip for embedding extraction                        │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                       │                                          │
│                                       ▼                                          │
│  3. TRANSCRIPTION (Whisper)                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Output: List[Word(text, start, end)]                                     │  │
│  │                                                                            │  │
│  │  Configuration:                                                            │  │
│  │  • Cloud: OpenAI Whisper API                                              │  │
│  │  • Local: faster-whisper (int8 quantized)                                 │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                       │                                          │
│                                       ▼                                          │
│  4. ALIGNMENT                                                                    │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Map Words to Segments based on timestamp overlap                         │  │
│  │                                                                            │  │
│  │  Output: List[Segment(text, start, end, speaker_id_cluster)]              │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                       │                                          │
│                                       ▼                                          │
│  5. IDENTIFICATION (Vector Search)                                              │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  For each speaker_id_cluster:                                              │  │
│  │    1. Extract audio bytes (segments ≥ 1.5s)                               │  │
│  │    2. Generate Embedding Vector (ECAPA-TDNN, 192-dim)                     │  │
│  │    3. Query DB: SELECT id, distance(embedding, E_new)                     │  │
│  │                 FROM speakers WHERE distance < 0.4                         │  │
│  │                                                                            │  │
│  │  Decision Logic:                                                           │  │
│  │  ┌─────────────────────────────────────────────────────────────────────┐  │  │
│  │  │  Similarity > 0.75  →  Auto-Label (High Confidence)                 │  │  │
│  │  │  Similarity 0.60-0.75  →  Suggest Label ("Is this Bill?")           │  │  │
│  │  │  Similarity < 0.60  →  Unknown Speaker (new temp ID)                │  │  │
│  │  └─────────────────────────────────────────────────────────────────────┘  │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                       │                                          │
│                                       ▼                                          │
│  6. STORAGE                                                                      │
│  ┌───────────────────────────────────────────────────────────────────────────┐  │
│  │  Save to SQLite:                                                           │  │
│  │  • Recording metadata                                                      │  │
│  │  • Segments with speaker assignments                                       │  │
│  │  • Embeddings for future matching                                         │  │
│  └───────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
└─────────────────────────────────────────────────────────────────────────────────┘
```

### 4.1 Retrospective Update

When a user labels a speaker, the system performs a **Retrospective Update**:

1. Creates/Updates Speaker entity with new embedding
2. Scans *all existing unknown segments* in the current recording
3. Auto-labels any segments matching the new profile > 0.75
4. Updates the speaker profile centroid

This ensures that labeling "Speaker A" as "Bill" at the end of a recording fixes all previous occurrences.

---

## 5. Database Schema (SQLite)

```sql
-- =====================================================
-- CORE TABLES
-- =====================================================

CREATE TABLE speakers (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    color_hex TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_unknown BOOLEAN DEFAULT 0,
    sample_count INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1  -- FALSE after merge/delete
);

-- Uses sqlite-vss virtual table for vector search
CREATE VIRTUAL TABLE speaker_embeddings USING vss0(
    embedding(192),
    speaker_id TEXT
);

-- Additional metadata table for embeddings
CREATE TABLE speaker_embedding_meta (
    id TEXT PRIMARY KEY,
    speaker_id TEXT NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    segment_id TEXT REFERENCES segments(id) ON DELETE SET NULL,
    quality_score REAL DEFAULT 1.0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE recordings (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    audio_path TEXT,
    duration_seconds REAL,
    processed_status TEXT DEFAULT 'pending'
        CHECK (processed_status IN ('pending', 'processing', 'completed', 'failed')),
    error_message TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE segments (
    id TEXT PRIMARY KEY,
    recording_id TEXT NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    speaker_id TEXT REFERENCES speakers(id) ON DELETE SET NULL,
    temp_speaker_id TEXT,  -- e.g., "Speaker A" before labeling
    start_time REAL NOT NULL,
    end_time REAL NOT NULL,
    text_content TEXT NOT NULL,
    embedding_blob BLOB,  -- Cached embedding for this segment
    confidence_score REAL,
    is_user_labeled BOOLEAN DEFAULT 0,
    has_overlap BOOLEAN DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    CHECK (end_time > start_time)
);

CREATE INDEX idx_segments_recording ON segments(recording_id);
CREATE INDEX idx_segments_speaker ON segments(speaker_id);
CREATE INDEX idx_segments_time ON segments(recording_id, start_time);

-- Full-text search for transcript search
CREATE VIRTUAL TABLE segments_fts USING fts5(
    text_content,
    content='segments',
    content_rowid='rowid',
    tokenize='porter unicode61'
);

-- Keep FTS in sync
CREATE TRIGGER segments_fts_insert AFTER INSERT ON segments BEGIN
    INSERT INTO segments_fts(rowid, text_content) VALUES (new.rowid, new.text_content);
END;

CREATE TRIGGER segments_fts_delete AFTER DELETE ON segments BEGIN
    INSERT INTO segments_fts(segments_fts, rowid, text_content)
    VALUES('delete', old.rowid, old.text_content);
END;

CREATE TRIGGER segments_fts_update AFTER UPDATE ON segments BEGIN
    INSERT INTO segments_fts(segments_fts, rowid, text_content)
    VALUES('delete', old.rowid, old.text_content);
    INSERT INTO segments_fts(rowid, text_content) VALUES (new.rowid, new.text_content);
END;
```

---

## 6. API Specification (Internal Sidecar)

The Python backend exposes a REST API (localhost only) for the Tauri frontend.

### 6.1 Security

```python
# Backend binds to 127.0.0.1 ONLY
# Shared secret token generated by Tauri on startup
# All requests must include: Authorization: Bearer <shared_secret>
```

### 6.2 Endpoints

**`POST /api/v1/process/ingest`**

* **Input:** File path, processing options (local/cloud).
* **Behavior:** Starts background task for transcription/diarization.
* **Response:** `{ "recording_id": "uuid", "status": "processing" }`

**`GET /api/v1/process/{recording_id}/status`**

* **Output:** `{ "status": "processing", "stage": "transcribing", "progress": 45 }`

**`POST /api/v1/speakers/label`**

* **Input:** `{ "segment_ids": [], "speaker_name": "Bill" }`
* **Behavior:**
  1. Creates/Updates Speaker entity.
  2. Extracts embeddings for provided `segment_ids`.
  3. Updates the Vector Index (re-calculates centroid).
  4. **Retrospective Update:** Scans *existing* unknown segments in the current recording and auto-labels them if they match the new profile > 0.75.
* **Response:** `{ "speaker_id": "uuid", "updated_segments": 12 }`

**`POST /api/v1/speakers/merge`**

* **Input:** `{ "target_speaker_id": "uuid", "source_speaker_id": "uuid" }`
* **Behavior:** Merges source into target, updates all segments.

**`GET /api/v1/recordings/{id}/transcript`**

* **Output:** JSON structured for the virtualized list UI.

**`POST /api/v1/recordings/{id}/export`**

* **Input:** `{ "format": "srt", "include_speakers": true }`
* **Output:** File download.

**`GET /api/v1/search`**

* **Input:** `?q=quarterly+report&speaker_id=uuid`
* **Output:** Matching segments with context.

---

## 7. Performance & Constraints

### 7.1 Resource Budget

| Resource | Constraint | Notes |
|----------|------------|-------|
| **Memory** | Max 2GB RAM | Critical for 8GB MacBooks |
| **CPU** | Yield to system threads | Whisper on MPS/Neural Engine if possible |
| **Disk** | AAC/M4A storage | ~10MB/hour vs 600MB/hour WAV |
| **Embeddings** | 768 bytes per embedding | Negligible storage impact |

### 7.2 Latency Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| **Transcription** | 0.3x real-time | 1hr audio → 20 min processing |
| **Speaker Search** | < 50ms | Match against 100 profiles |
| **UI Response** | < 100ms | All interactions |
| **Search** | < 100ms | Full-text query |

### 7.3 Minimum Requirements

* **macOS:** 13.0 (Ventura) or later
* **Processor:** Apple Silicon (M1/M2/M3) strongly recommended
* **Memory:** 8GB minimum, 16GB recommended
* **Disk:** 500MB for app + models

---

## 8. Security & Privacy

### 8.1 Local API Security

```python
# The Python API must:
# 1. Bind STRICTLY to 127.0.0.1 (no 0.0.0.0)
# 2. Require shared secret token in Authorization header
# 3. Token generated by Tauri on each app startup
# 4. Reject all requests without valid token
```

### 8.2 Data Storage

* **Location:** `~/Library/Application Support/Scribe/`
* **Audio:** Stored as AAC/M4A (compressed, space-efficient)
* **Database:** Single SQLite file with embeddings
* **No Cloud:** All data stays local unless user explicitly exports

### 8.3 Required Permissions

```xml
<!-- Info.plist -->
<key>NSScreenCaptureUsageDescription</key>
<string>Scribe needs screen recording access to capture meeting audio.</string>

<key>NSMicrophoneUsageDescription</key>
<string>Scribe needs microphone access to record your voice.</string>
```

---

## 9. Development Milestones

### Phase 1: The "Manual" MVP

* **Goal:** User records, sees "Speaker 0/1", and manually renames them. No learning yet.
* **Deliverables:**
  - [ ] Tauri + FastAPI sidecar glue
  - [ ] Whisper + Pyannote pipeline working on files
  - [ ] Basic Transcript UI with virtualized list
  - [ ] File import (MP3, WAV, M4A, MP4)
  - [ ] Export to TXT, SRT

### Phase 2: The Learning Loop (Current Target)

* **Goal:** "Label once, recognize forever."
* **Deliverables:**
  - [ ] Vector database integration (`sqlite-vss`)
  - [ ] ECAPA-TDNN embedding extraction pipeline
  - [ ] Speaker matching logic with confidence thresholds
  - [ ] Retrospective updating (fixing the current document based on a new label)
  - [ ] UI distinction: Auto-labeled (dotted) vs Manually labeled (solid)
  - [ ] Minimum 3 samples requirement before auto-labeling

### Phase 3: Polish & Performance

* **Goal:** Native ScreenCaptureKit and Optimization.
* **Deliverables:**
  - [ ] Replace ffmpeg capture with Rust `ScreenCaptureKit` implementation
  - [ ] Optimize Whisper quantization (int8)
  - [ ] DeepFilterNet preprocessing for low-quality audio
  - [ ] Export features (VTT, JSON)
  - [ ] Merge Speakers UI
  - [ ] PyInstaller bundle for Python sidecar

---

## 10. Devil's Advocate: Warnings & Risk Mitigation

### 10.1 Risk: "The Hallucinating Diarizer"

* **Issue:** Pyannote might split one person into "Speaker A" and "Speaker B" if their tone changes (e.g., whispering vs. shouting).
* **Mitigation:**
  - Implement a "Merge Speakers" UI function
  - The embedding system accepts *diverse* samples into one profile
  - Centroid is updated as weighted average, not replaced

### 10.2 Risk: Poor Audio Quality (Meeting Room Echo)

* **Issue:** Reverb destroys diarization accuracy.
* **Mitigation:**
  - Pre-processing step using `DeepFilterNet` for audio cleanup
  - If SNR < 15dB, warn user: "Audio quality too low for reliable speaker ID"
  - Still allow transcription, just skip speaker matching

### 10.3 Risk: Profile Drift

* **Issue:** A user labeled 6 months ago might sound different today.
* **Mitigation:**
  - **Moving Average Centroid:** When a speaker is identified with High Confidence (>0.85), slowly update their profile embedding with new data
  - Weight: 0.05 for new data (centroid_new = 0.95 * centroid_old + 0.05 * new_embedding)
  - Keeps profile fresh without allowing noise to corrupt it

### 10.4 Risk: Python Dependency on macOS

* **Issue:** Python environments are fragile.
* **Mitigation:**
  - Use PyInstaller to bundle the backend as a frozen binary
  - Do *not* rely on system Python
  - All dependencies vendored into the app bundle

### 10.5 Risk: Overlapping Speech Misattribution

* **Issue:** System might assign overlapping speech to wrong speaker.
* **Mitigation:**
  - Overlapping speech > 500ms marked as "Multiple Speakers"
  - Never extract embeddings from overlapping segments
  - UI shows overlap indicator

---

## Appendix A: Speaker Matching Thresholds

| Similarity Score | Action | UI Treatment |
|-----------------|--------|--------------|
| **≥ 0.75** | Auto-assign speaker | Solid color bar, auto-labeled badge |
| **0.60 - 0.74** | Suggest match | Dashed border, "Is this Bill?" prompt |
| **< 0.60** | New unknown speaker | Orange color, "Label" button |

## Appendix B: Minimum Requirements for Speaker Profile

| Requirement | Minimum | Recommended |
|-------------|---------|-------------|
| Labeled segments | 3 | 5 |
| Segment duration | 1.5 seconds | 3 seconds |
| Total audio | 5 seconds | 15 seconds |
| Audio quality (SNR) | 10 dB | 20 dB |

## Appendix C: File Formats

| Format | Use Case | Extension |
|--------|----------|-----------|
| Audio storage | Recordings | .m4a (AAC) |
| Database | All structured data | .db (SQLite) |
| Export: Plain text | Simple transcripts | .txt |
| Export: Subtitles | Video editing | .srt, .vtt |
| Export: Data | Integration | .json |

---

*End of Specification*
