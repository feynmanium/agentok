# Mac Screen Transcription App - Product Specification

## Project Codename: "Scribe"

**Version:** 1.0.0
**Status:** Draft
**Author:** Claude
**Date:** 2025-01-25

---

## 1. Executive Summary

### 1.1 Product Vision
Build an Otter.ai-like transcription application for macOS that captures audio from screen recordings or live system audio, transcribes speech in real-time, performs speaker diarization, and learns speaker identities from user-provided labels for automatic speaker recognition in future sessions.

### 1.2 Core Value Proposition
- **Real-time transcription** from Mac screen recordings and system audio
- **Automatic speaker separation** (diarization)
- **Learning speaker identification** - manually label once, auto-identify forever
- **Voice fingerprinting** - builds speaker profiles from labeled segments
- **Searchable transcript archive** with speaker-attributed content

### 1.3 Key Differentiators
1. **Local-first processing** - privacy-focused, runs on-device where possible
2. **Hybrid cloud** - optional cloud processing for better accuracy
3. **Incremental learning** - speaker recognition improves with each correction
4. **Mac-native** - deep integration with macOS screen capture APIs

---

## 2. User Personas & Use Cases

### 2.1 Primary Personas

#### Persona 1: Meeting Professional (Bill)
- Records Zoom/Teams/Meet calls daily
- Needs accurate speaker attribution for meeting notes
- Values searchable transcript history
- Wants to share meeting summaries with team

#### Persona 2: Content Creator (Gill)
- Records podcasts and interviews
- Needs to identify multiple guests across episodes
- Exports transcripts for show notes
- Wants speaker-labeled timestamps for editing

#### Persona 3: Researcher (Alex)
- Records user research interviews
- Needs precise speaker attribution for analysis
- Exports to qualitative analysis tools
- Wants to track themes across multiple participants

### 2.2 Core Use Cases

| Use Case | Description | Priority |
|----------|-------------|----------|
| UC-1 | Capture system audio during screen recording | P0 |
| UC-2 | Real-time transcription with speaker labels | P0 |
| UC-3 | Manual speaker labeling (Speaker A → "Bill") | P0 |
| UC-4 | Auto-identification of known speakers | P0 |
| UC-5 | Search transcripts by speaker and content | P1 |
| UC-6 | Export transcripts (TXT, SRT, JSON, DOCX) | P1 |
| UC-7 | Import existing audio/video files | P1 |
| UC-8 | Live transcription mode (no recording) | P2 |
| UC-9 | Real-time collaborative viewing | P2 |
| UC-10 | Summary generation with speaker highlights | P2 |

---

## 3. System Architecture

### 3.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           Mac Desktop Application                            │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │  Screen/Audio   │  │   Transcription │  │      Speaker Learning       │  │
│  │    Capture      │  │     Engine      │  │         Engine              │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  ┌───────────────────────┐  │  │
│  │  │ScreenCap │  │  │  │ Whisper   │  │  │  │  Voice Embedding      │  │  │
│  │  │   API    │  │  │  │  (local)  │  │  │  │    Extractor          │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  │  (pyannote/ecapa)     │  │  │
│  │  ┌───────────┐  │  │  ┌───────────┐  │  │  └───────────────────────┘  │  │
│  │  │ CoreAudio │  │  │  │ Diariza- │  │  │  ┌───────────────────────┐  │  │
│  │  │  Capture  │  │  │  │   tion   │  │  │  │  Speaker Profile      │  │  │
│  │  └───────────┘  │  │  │(pyannote)│  │  │  │    Database           │  │  │
│  │  ┌───────────┐  │  │  └───────────┘  │  │  └───────────────────────┘  │  │
│  │  │ Audio     │  │  │  ┌───────────┐  │  │  ┌───────────────────────┐  │  │
│  │  │ Buffer    │  │  │  │ Alignment │  │  │  │  Similarity Matcher   │  │  │
│  │  └───────────┘  │  │  └───────────┘  │  │  │  (cosine distance)    │  │  │
│  └─────────────────┘  └─────────────────┘  │  └───────────────────────┘  │  │
│                                            └─────────────────────────────┘  │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │                           Local SQLite + Vector DB                       ││
│  │   ┌──────────────┐  ┌──────────────┐  ┌───────────────────────────────┐ ││
│  │   │  Transcripts │  │   Speakers   │  │     Voice Embeddings          │ ││
│  │   │   (FTS5)     │  │   Profiles   │  │     (sqlite-vss)              │ ││
│  │   └──────────────┘  └──────────────┘  └───────────────────────────────┘ ││
│  └─────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      │ Optional Cloud Sync
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                        Cloud Backend (Optional)                              │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────────────────┐  │
│  │  Supabase       │  │  Enhanced       │  │    Speaker Profile          │  │
│  │  PostgreSQL     │  │  Transcription  │  │    Cross-device Sync        │  │
│  │  + pgvector     │  │  (Whisper API)  │  │                             │  │
│  └─────────────────┘  └─────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 3.2 Component Architecture

#### 3.2.1 Audio Capture Layer
```
┌─────────────────────────────────────────────────────┐
│               Audio Capture Manager                  │
├─────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────────────┐   │
│  │ SCStreamAudio   │  │ AVCaptureSession        │   │
│  │ (macOS 13+)     │  │ (fallback)              │   │
│  │                 │  │                         │   │
│  │ - System audio  │  │ - Microphone input      │   │
│  │ - App-specific  │  │ - External devices      │   │
│  │ - Screen audio  │  │                         │   │
│  └────────┬────────┘  └───────────┬─────────────┘   │
│           │                       │                  │
│           └───────────┬───────────┘                  │
│                       ▼                              │
│           ┌─────────────────────┐                    │
│           │   Audio Buffer      │                    │
│           │  (Ring Buffer)      │                    │
│           │  - 16kHz mono       │                    │
│           │  - Float32          │                    │
│           │  - 30sec window     │                    │
│           └─────────────────────┘                    │
└─────────────────────────────────────────────────────┘
```

#### 3.2.2 Transcription Pipeline
```
Audio Buffer
     │
     ▼
┌─────────────────┐
│  VAD (Voice     │ ───▶ Silence Detection
│  Activity Det.) │      Skip non-speech segments
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Speaker        │ ───▶ pyannote.audio 3.1
│  Diarization    │      Segment by speaker
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌─────────────────────────┐
│  Whisper        │ ◀───│  Parallel Processing    │
│  Transcription  │     │  (per speaker segment)  │
└────────┬────────┘     └─────────────────────────┘
         │
         ▼
┌─────────────────┐
│  Speaker        │ ───▶ Match against known
│  Identification │      speaker profiles
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Transcript     │ ───▶ Store with timestamps,
│  Assembly       │      speaker IDs, embeddings
└─────────────────┘
```

#### 3.2.3 Speaker Learning System
```
┌─────────────────────────────────────────────────────────────────┐
│                    Speaker Learning Pipeline                     │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  1. VOICE EMBEDDING EXTRACTION                           │    │
│  │                                                          │    │
│  │  Audio Segment ──▶ ECAPA-TDNN ──▶ 192-dim Embedding     │    │
│  │                    (speechbrain)                         │    │
│  │                                                          │    │
│  │  Alternative: pyannote/embedding or resemblyzer          │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  2. SPEAKER PROFILE MANAGEMENT                           │    │
│  │                                                          │    │
│  │  On Manual Label ("Speaker A" → "Bill"):                 │    │
│  │                                                          │    │
│  │  ┌─────────────┐     ┌──────────────────────────────┐   │    │
│  │  │ Labeled     │     │     Speaker Profile           │   │    │
│  │  │ Segments    │ ──▶ │     ┌─────────────────────┐  │   │    │
│  │  │ for "Bill"  │     │     │ embedding_centroid  │  │   │    │
│  │  └─────────────┘     │     │ (avg of embeddings) │  │   │    │
│  │                      │     ├─────────────────────┤  │   │    │
│  │                      │     │ embedding_samples[] │  │   │    │
│  │                      │     │ (up to 50 samples)  │  │   │    │
│  │                      │     ├─────────────────────┤  │   │    │
│  │                      │     │ confidence_score    │  │   │    │
│  │                      │     │ sample_count        │  │   │    │
│  │                      │     └─────────────────────┘  │   │    │
│  │                      └──────────────────────────────┘   │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  3. SPEAKER MATCHING (Future Sessions)                   │    │
│  │                                                          │    │
│  │  Unknown Speaker Embedding                               │    │
│  │          │                                               │    │
│  │          ▼                                               │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │  Cosine Similarity vs All Known Profiles        │    │    │
│  │  │                                                  │    │    │
│  │  │  similarity = dot(emb_unknown, emb_profile)     │    │    │
│  │  │              / (norm(emb_unknown) * norm(prof)) │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  │          │                                               │    │
│  │          ▼                                               │    │
│  │  ┌─────────────────────────────────────────────────┐    │    │
│  │  │  Decision Logic:                                 │    │    │
│  │  │                                                  │    │    │
│  │  │  if max_similarity > 0.75:                       │    │    │
│  │  │      assign known_speaker (high confidence)      │    │    │
│  │  │  elif max_similarity > 0.60:                     │    │    │
│  │  │      suggest known_speaker (needs confirmation)  │    │    │
│  │  │  else:                                           │    │    │
│  │  │      assign new temporary ID ("Speaker C")       │    │    │
│  │  └─────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│                              ▼                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  4. CONTINUOUS LEARNING                                  │    │
│  │                                                          │    │
│  │  On Each Confirmation/Correction:                        │    │
│  │  - Add new embedding to speaker's sample pool            │    │
│  │  - Recompute centroid (rolling average)                  │    │
│  │  - Prune old/outlier samples if pool > 50                │    │
│  │  - Update confidence score based on match history        │    │
│  │                                                          │    │
│  │  Profile Evolution:                                      │    │
│  │  centroid_new = α * centroid_old + (1-α) * new_embedding │    │
│  │  where α = 0.95 (slow adaptation to prevent drift)       │    │
│  └─────────────────────────────────────────────────────────┘    │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Models

### 4.1 Database Schema

```sql
-- =====================================================
-- CORE TABLES
-- =====================================================

-- Recording sessions (one per screen recording)
CREATE TABLE recordings (
    id              TEXT PRIMARY KEY,          -- UUID
    title           TEXT NOT NULL,
    description     TEXT,
    source_type     TEXT NOT NULL,             -- 'screen_capture', 'audio_file', 'live_mic'
    source_path     TEXT,                      -- Path to original media file (if any)
    duration_ms     INTEGER,
    status          TEXT NOT NULL DEFAULT 'processing',  -- 'processing', 'completed', 'failed'
    audio_format    TEXT,                      -- 'wav', 'mp3', 'aac'
    sample_rate     INTEGER DEFAULT 16000,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata        TEXT                       -- JSON blob for extensibility
);

-- Speaker profiles (learned from user labels)
CREATE TABLE speakers (
    id              TEXT PRIMARY KEY,          -- UUID
    name            TEXT NOT NULL,             -- User-assigned name ("Bill", "Gill")
    email           TEXT,                      -- Optional contact info
    avatar_path     TEXT,                      -- Optional profile picture
    embedding_centroid BLOB,                   -- 192-dim float32 vector (primary)
    sample_count    INTEGER DEFAULT 0,
    confidence      REAL DEFAULT 0.0,          -- 0.0 to 1.0, improves with samples
    first_seen_at   DATETIME,
    last_seen_at    DATETIME,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata        TEXT                       -- JSON blob
);

-- Voice embedding samples for each speaker (for retraining)
CREATE TABLE speaker_embeddings (
    id              TEXT PRIMARY KEY,
    speaker_id      TEXT NOT NULL REFERENCES speakers(id) ON DELETE CASCADE,
    embedding       BLOB NOT NULL,             -- 192-dim float32 vector
    source_segment_id TEXT REFERENCES segments(id),
    quality_score   REAL DEFAULT 1.0,          -- Audio quality indicator
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_speaker_embeddings_speaker ON speaker_embeddings(speaker_id);

-- Transcript segments (individual utterances)
CREATE TABLE segments (
    id              TEXT PRIMARY KEY,
    recording_id    TEXT NOT NULL REFERENCES recordings(id) ON DELETE CASCADE,
    speaker_id      TEXT REFERENCES speakers(id),          -- NULL if unidentified
    temp_speaker_id TEXT,                      -- Temporary ID before labeling ("Speaker A")
    text            TEXT NOT NULL,
    start_ms        INTEGER NOT NULL,
    end_ms          INTEGER NOT NULL,
    confidence      REAL,                      -- Whisper confidence
    speaker_confidence REAL,                   -- Speaker ID confidence
    is_user_verified BOOLEAN DEFAULT FALSE,    -- User confirmed speaker
    embedding       BLOB,                      -- Voice embedding for this segment
    word_timestamps TEXT,                      -- JSON array of word-level timing
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_segments_recording ON segments(recording_id);
CREATE INDEX idx_segments_speaker ON segments(speaker_id);
CREATE INDEX idx_segments_time ON segments(recording_id, start_ms);

-- Full-text search virtual table
CREATE VIRTUAL TABLE segments_fts USING fts5(
    text,
    content='segments',
    content_rowid='rowid'
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

-- Vector similarity search for speaker matching (using sqlite-vss or hnswlib)
-- This is a conceptual representation - actual implementation depends on vector DB choice
CREATE VIRTUAL TABLE speaker_vectors USING vss0(
    embedding(192)  -- 192-dimensional ECAPA-TDNN embeddings
);

-- =====================================================
-- SETTINGS & METADATA
-- =====================================================

CREATE TABLE settings (
    key             TEXT PRIMARY KEY,
    value           TEXT NOT NULL,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Default settings
INSERT INTO settings (key, value) VALUES
    ('transcription_model', 'whisper-large-v3'),
    ('transcription_language', 'auto'),
    ('diarization_enabled', 'true'),
    ('speaker_matching_threshold', '0.75'),
    ('speaker_suggestion_threshold', '0.60'),
    ('auto_punctuation', 'true'),
    ('profanity_filter', 'false');

-- =====================================================
-- EXPORT HISTORY
-- =====================================================

CREATE TABLE exports (
    id              TEXT PRIMARY KEY,
    recording_id    TEXT NOT NULL REFERENCES recordings(id),
    format          TEXT NOT NULL,             -- 'txt', 'srt', 'vtt', 'json', 'docx'
    file_path       TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### 4.2 TypeScript/Pydantic Models

```typescript
// ============================================
// TypeScript Models (Frontend)
// ============================================

interface Recording {
  id: string;
  title: string;
  description?: string;
  sourceType: 'screen_capture' | 'audio_file' | 'live_mic';
  sourcePath?: string;
  durationMs: number;
  status: 'processing' | 'completed' | 'failed';
  createdAt: Date;
  updatedAt: Date;
  segments?: Segment[];
  speakerSummary?: SpeakerSummary[];
}

interface Speaker {
  id: string;
  name: string;
  email?: string;
  avatarPath?: string;
  sampleCount: number;
  confidence: number;
  firstSeenAt: Date;
  lastSeenAt: Date;
  color?: string;  // UI display color
}

interface Segment {
  id: string;
  recordingId: string;
  speakerId?: string;
  tempSpeakerId?: string;
  text: string;
  startMs: number;
  endMs: number;
  confidence: number;
  speakerConfidence: number;
  isUserVerified: boolean;
  wordTimestamps?: WordTimestamp[];
  speaker?: Speaker;
}

interface WordTimestamp {
  word: string;
  startMs: number;
  endMs: number;
  confidence: number;
}

interface SpeakerSummary {
  speakerId: string;
  speakerName: string;
  totalDurationMs: number;
  segmentCount: number;
  wordCount: number;
  percentage: number;
}

interface SpeakerMatch {
  speakerId: string;
  speakerName: string;
  similarity: number;
  confidence: 'high' | 'medium' | 'low';
}

interface TranscriptionProgress {
  recordingId: string;
  stage: 'capturing' | 'diarizing' | 'transcribing' | 'identifying' | 'complete';
  progress: number;  // 0-100
  currentSegment?: number;
  totalSegments?: number;
}
```

```python
# ============================================
# Pydantic Models (Backend)
# ============================================

from pydantic import BaseModel, Field
from datetime import datetime
from typing import Optional, List, Literal
from enum import Enum
import numpy as np

class SourceType(str, Enum):
    SCREEN_CAPTURE = "screen_capture"
    AUDIO_FILE = "audio_file"
    LIVE_MIC = "live_mic"

class RecordingStatus(str, Enum):
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"

class Recording(BaseModel):
    id: str
    title: str
    description: Optional[str] = None
    source_type: SourceType
    source_path: Optional[str] = None
    duration_ms: Optional[int] = None
    status: RecordingStatus = RecordingStatus.PROCESSING
    audio_format: Optional[str] = None
    sample_rate: int = 16000
    created_at: datetime = Field(default_factory=datetime.utcnow)
    updated_at: datetime = Field(default_factory=datetime.utcnow)
    metadata: Optional[dict] = None

class Speaker(BaseModel):
    id: str
    name: str
    email: Optional[str] = None
    avatar_path: Optional[str] = None
    sample_count: int = 0
    confidence: float = 0.0
    first_seen_at: Optional[datetime] = None
    last_seen_at: Optional[datetime] = None
    created_at: datetime = Field(default_factory=datetime.utcnow)

class VoiceEmbedding(BaseModel):
    """192-dimensional voice embedding from ECAPA-TDNN"""
    speaker_id: str
    embedding: List[float]  # 192 floats
    quality_score: float = 1.0
    source_segment_id: Optional[str] = None

class Segment(BaseModel):
    id: str
    recording_id: str
    speaker_id: Optional[str] = None
    temp_speaker_id: Optional[str] = None
    text: str
    start_ms: int
    end_ms: int
    confidence: float
    speaker_confidence: Optional[float] = None
    is_user_verified: bool = False
    word_timestamps: Optional[List[dict]] = None

class SpeakerLabelRequest(BaseModel):
    """Request to label a temporary speaker with a real identity"""
    segment_ids: List[str]  # Segments to label
    temp_speaker_id: str  # e.g., "Speaker A"
    speaker_id: Optional[str] = None  # Existing speaker ID
    new_speaker_name: Optional[str] = None  # Create new speaker

class SpeakerMatchResult(BaseModel):
    speaker_id: str
    speaker_name: str
    similarity: float
    confidence: Literal["high", "medium", "low"]

class TranscriptionConfig(BaseModel):
    language: str = "auto"
    model: str = "whisper-large-v3"
    enable_diarization: bool = True
    enable_speaker_matching: bool = True
    speaker_matching_threshold: float = 0.75
    speaker_suggestion_threshold: float = 0.60
    enable_punctuation: bool = True
    enable_word_timestamps: bool = True
```

---

## 5. API Specification

### 5.1 REST API Endpoints

```yaml
openapi: 3.0.3
info:
  title: Scribe Transcription API
  version: 1.0.0

paths:
  # ==========================================
  # RECORDINGS
  # ==========================================

  /api/v1/recordings:
    get:
      summary: List all recordings
      parameters:
        - name: limit
          in: query
          schema: { type: integer, default: 50 }
        - name: offset
          in: query
          schema: { type: integer, default: 0 }
        - name: status
          in: query
          schema: { type: string, enum: [processing, completed, failed] }
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  recordings: { type: array, items: { $ref: '#/components/schemas/Recording' } }
                  total: { type: integer }

    post:
      summary: Create new recording (start capture or upload file)
      requestBody:
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                title: { type: string }
                source_type: { type: string, enum: [screen_capture, audio_file, live_mic] }
                audio_file: { type: string, format: binary }
                config: { $ref: '#/components/schemas/TranscriptionConfig' }
      responses:
        201:
          content:
            application/json:
              schema: { $ref: '#/components/schemas/Recording' }

  /api/v1/recordings/{id}:
    get:
      summary: Get recording details with segments
      responses:
        200:
          content:
            application/json:
              schema:
                allOf:
                  - $ref: '#/components/schemas/Recording'
                  - type: object
                    properties:
                      segments: { type: array, items: { $ref: '#/components/schemas/Segment' } }
                      speaker_summary: { type: array, items: { $ref: '#/components/schemas/SpeakerSummary' } }

    patch:
      summary: Update recording metadata
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                title: { type: string }
                description: { type: string }

    delete:
      summary: Delete recording and all segments

  /api/v1/recordings/{id}/export:
    post:
      summary: Export transcript in specified format
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                format: { type: string, enum: [txt, srt, vtt, json, docx] }
                include_timestamps: { type: boolean, default: true }
                include_speaker_names: { type: boolean, default: true }
      responses:
        200:
          content:
            application/octet-stream: {}

  # ==========================================
  # LIVE TRANSCRIPTION (WebSocket)
  # ==========================================

  /api/v1/recordings/{id}/stream:
    get:
      summary: WebSocket endpoint for live transcription updates
      description: |
        Real-time stream of transcription progress and segments.

        Messages from server:
        - { type: "progress", data: TranscriptionProgress }
        - { type: "segment", data: Segment }
        - { type: "speaker_detected", data: { temp_speaker_id, suggested_matches: SpeakerMatch[] } }
        - { type: "complete", data: { recording_id } }
        - { type: "error", data: { message } }

  # ==========================================
  # SEGMENTS
  # ==========================================

  /api/v1/recordings/{recording_id}/segments:
    get:
      summary: Get all segments for a recording
      parameters:
        - name: speaker_id
          in: query
          description: Filter by speaker
        - name: start_ms
          in: query
          description: Filter segments after this time
        - name: end_ms
          in: query
          description: Filter segments before this time

  /api/v1/segments/{id}:
    patch:
      summary: Update segment (edit text, assign speaker)
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                text: { type: string }
                speaker_id: { type: string }
                is_user_verified: { type: boolean }

  /api/v1/segments/bulk-label:
    post:
      summary: Label multiple segments with a speaker identity
      description: |
        Used when user labels "Speaker A" as "Bill". Updates all segments
        with temp_speaker_id and triggers speaker learning.
      requestBody:
        content:
          application/json:
            schema: { $ref: '#/components/schemas/SpeakerLabelRequest' }
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  updated_count: { type: integer }
                  speaker: { $ref: '#/components/schemas/Speaker' }

  # ==========================================
  # SPEAKERS
  # ==========================================

  /api/v1/speakers:
    get:
      summary: List all known speakers
      responses:
        200:
          content:
            application/json:
              schema:
                type: array
                items: { $ref: '#/components/schemas/Speaker' }

    post:
      summary: Create new speaker profile
      requestBody:
        content:
          application/json:
            schema:
              type: object
              required: [name]
              properties:
                name: { type: string }
                email: { type: string }

  /api/v1/speakers/{id}:
    get:
      summary: Get speaker details with statistics
    patch:
      summary: Update speaker profile
    delete:
      summary: Delete speaker (segments become unassigned)

  /api/v1/speakers/{id}/merge:
    post:
      summary: Merge another speaker into this one
      description: Combines voice profiles and reassigns all segments
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                source_speaker_id: { type: string }

  # ==========================================
  # SEARCH
  # ==========================================

  /api/v1/search:
    get:
      summary: Full-text search across all transcripts
      parameters:
        - name: q
          in: query
          required: true
          schema: { type: string }
        - name: speaker_id
          in: query
          description: Filter by speaker
        - name: recording_id
          in: query
          description: Filter by recording
        - name: date_from
          in: query
          schema: { type: string, format: date }
        - name: date_to
          in: query
          schema: { type: string, format: date }
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  results:
                    type: array
                    items:
                      type: object
                      properties:
                        segment: { $ref: '#/components/schemas/Segment' }
                        recording: { $ref: '#/components/schemas/Recording' }
                        highlight: { type: string }
                  total: { type: integer }

  # ==========================================
  # CAPTURE CONTROL (Mac-specific)
  # ==========================================

  /api/v1/capture/start:
    post:
      summary: Start screen/audio capture
      requestBody:
        content:
          application/json:
            schema:
              type: object
              properties:
                title: { type: string }
                capture_type: { type: string, enum: [screen_audio, mic_only, both] }
                target_app: { type: string, description: "Bundle ID for app-specific capture" }
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  recording_id: { type: string }
                  status: { type: string }

  /api/v1/capture/stop:
    post:
      summary: Stop active capture
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  recording_id: { type: string }
                  duration_ms: { type: integer }

  /api/v1/capture/status:
    get:
      summary: Get current capture status
      responses:
        200:
          content:
            application/json:
              schema:
                type: object
                properties:
                  is_capturing: { type: boolean }
                  recording_id: { type: string }
                  duration_ms: { type: integer }
                  audio_level: { type: number }

components:
  schemas:
    Recording:
      type: object
      properties:
        id: { type: string, format: uuid }
        title: { type: string }
        description: { type: string }
        source_type: { type: string, enum: [screen_capture, audio_file, live_mic] }
        duration_ms: { type: integer }
        status: { type: string, enum: [processing, completed, failed] }
        created_at: { type: string, format: date-time }
        updated_at: { type: string, format: date-time }

    Speaker:
      type: object
      properties:
        id: { type: string, format: uuid }
        name: { type: string }
        email: { type: string }
        avatar_path: { type: string }
        sample_count: { type: integer }
        confidence: { type: number }
        first_seen_at: { type: string, format: date-time }
        last_seen_at: { type: string, format: date-time }

    Segment:
      type: object
      properties:
        id: { type: string, format: uuid }
        recording_id: { type: string }
        speaker_id: { type: string }
        temp_speaker_id: { type: string }
        text: { type: string }
        start_ms: { type: integer }
        end_ms: { type: integer }
        confidence: { type: number }
        speaker_confidence: { type: number }
        is_user_verified: { type: boolean }
        word_timestamps:
          type: array
          items:
            type: object
            properties:
              word: { type: string }
              start_ms: { type: integer }
              end_ms: { type: integer }

    SpeakerSummary:
      type: object
      properties:
        speaker_id: { type: string }
        speaker_name: { type: string }
        total_duration_ms: { type: integer }
        segment_count: { type: integer }
        word_count: { type: integer }
        percentage: { type: number }

    SpeakerMatch:
      type: object
      properties:
        speaker_id: { type: string }
        speaker_name: { type: string }
        similarity: { type: number }
        confidence: { type: string, enum: [high, medium, low] }

    SpeakerLabelRequest:
      type: object
      required: [segment_ids, temp_speaker_id]
      properties:
        segment_ids: { type: array, items: { type: string } }
        temp_speaker_id: { type: string }
        speaker_id: { type: string }
        new_speaker_name: { type: string }

    TranscriptionConfig:
      type: object
      properties:
        language: { type: string, default: "auto" }
        model: { type: string, default: "whisper-large-v3" }
        enable_diarization: { type: boolean, default: true }
        enable_speaker_matching: { type: boolean, default: true }
        speaker_matching_threshold: { type: number, default: 0.75 }
```

---

## 6. User Interface Specification

### 6.1 Screen Layouts

#### 6.1.1 Main Dashboard
```
┌─────────────────────────────────────────────────────────────────────────────┐
│  [Logo] Scribe                              [Search...]     [Settings] [?]  │
├───────────────────────────────┬─────────────────────────────────────────────┤
│                               │                                             │
│  ┌─────────────────────────┐  │  Recent Recordings                          │
│  │                         │  │  ─────────────────                          │
│  │    [▶] Start Recording  │  │                                             │
│  │                         │  │  ┌─────────────────────────────────────────┐│
│  │    Screen + Audio       │  │  │ 📹 Team Standup - Jan 25              ││
│  │    ○ Microphone Only    │  │  │    Duration: 45:23 | Speakers: 4       ││
│  │    ○ System Audio Only  │  │  │    Bill (35%) • Gill (28%) • Alex (22%)││
│  │                         │  │  └─────────────────────────────────────────┘│
│  └─────────────────────────┘  │                                             │
│                               │  ┌─────────────────────────────────────────┐│
│  ┌─────────────────────────┐  │  │ 📹 Client Interview - Jan 24          ││
│  │  Drop files here        │  │  │    Duration: 1:23:45 | Speakers: 2     ││
│  │  or click to upload     │  │  │    Bill (60%) • New Speaker (40%)      ││
│  │                         │  │  └─────────────────────────────────────────┘│
│  │  Supports: MP3, WAV,    │  │                                             │
│  │  M4A, MP4, WebM         │  │  ┌─────────────────────────────────────────┐│
│  └─────────────────────────┘  │  │ 🎤 Podcast Episode 42 - Jan 23        ││
│                               │  │    Duration: 58:12 | Speakers: 3       ││
│  ───────────────────────────  │  └─────────────────────────────────────────┘│
│                               │                                             │
│  Known Speakers (12)          │                                             │
│  ┌────┐ ┌────┐ ┌────┐        │  [Load More...]                             │
│  │Bill│ │Gill│ │Alex│        │                                             │
│  └────┘ └────┘ └────┘        │                                             │
│  [View All Speakers...]       │                                             │
│                               │                                             │
└───────────────────────────────┴─────────────────────────────────────────────┘
```

#### 6.1.2 Transcript View with Speaker Labeling
```
┌─────────────────────────────────────────────────────────────────────────────┐
│  ← Back    Team Standup - Jan 25, 2025                   [Export ▼] [Share] │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │  ▶ ──────●────────────────────────────────────────────── 12:34 / 45:23 ││
│  └─────────────────────────────────────────────────────────────────────────┘│
│                                                                             │
├────────────────────────────────────────────┬────────────────────────────────┤
│                                            │                                │
│  Speakers in this recording:               │  [Search transcript...]        │
│                                            │                                │
│  ┌──────────────────────────────────────┐  │  ┌────────────────────────────┐│
│  │ ● Bill (identified)         35%     │  │  │ 00:00:12  Bill             ││
│  │   12 segments • 15:52 speaking      │  │  │ Alright, let's get started ││
│  └──────────────────────────────────────┘  │  │ with today's standup.      ││
│                                            │  └────────────────────────────┘│
│  ┌──────────────────────────────────────┐  │                                │
│  │ ● Gill (identified)         28%     │  │  ┌────────────────────────────┐│
│  │   8 segments • 12:41 speaking       │  │  │ 00:00:18  Gill             ││
│  └──────────────────────────────────────┘  │  │ Sure. Yesterday I finished ││
│                                            │  │ the API integration.       ││
│  ┌──────────────────────────────────────┐  │  └────────────────────────────┘│
│  │ ○ Speaker C (unidentified)  22%     │  │                                │
│  │   6 segments • 9:58 speaking        │  │  ┌────────────────────────────┐│
│  │   ┌─────────────────────────────┐   │  │  │ 00:00:35  Speaker C  ⚠️   ││
│  │   │ Looks like: Alex (78%)     │   │  │  │ I've been working on the   ││
│  │   │            [Confirm Alex]  │   │  │  │ frontend components.       ││
│  │   │            [Different...]  │   │  │  │ ┌──────────────────────┐   ││
│  │   └─────────────────────────────┘   │  │  │ │ Click to label speaker│  ││
│  └──────────────────────────────────────┘  │  │ │ ○ Alex (suggested)   │  ││
│                                            │  │ │ ○ New speaker...     │  ││
│  ┌──────────────────────────────────────┐  │  │ └──────────────────────┘   ││
│  │ ○ Speaker D (unidentified)  15%     │  │  └────────────────────────────┘│
│  │   4 segments • 6:52 speaking        │  │                                │
│  │   [Label as...]                     │  │  ┌────────────────────────────┐│
│  └──────────────────────────────────────┘  │  │ 00:01:12  Bill             ││
│                                            │  │ Great progress. Any        ││
│                                            │  │ blockers?                  ││
│                                            │  └────────────────────────────┘│
│                                            │                                │
└────────────────────────────────────────────┴────────────────────────────────┘
```

#### 6.1.3 Speaker Labeling Modal
```
┌─────────────────────────────────────────────────────────────┐
│  Label Speaker                                          [X] │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Who is "Speaker C"?                                        │
│                                                             │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ 🔊 "I've been working on the frontend components..."   ││
│  │    [▶ Play Sample]                                     ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
│  Suggested Matches:                                         │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ ◉ Alex Chen                              78% match     ││
│  │   Last seen: 3 days ago • 45 recordings               ││
│  ├─────────────────────────────────────────────────────────┤│
│  │ ○ Jordan Smith                           62% match     ││
│  │   Last seen: 1 week ago • 12 recordings               ││
│  └─────────────────────────────────────────────────────────┘│
│                                                             │
│  ─────────────── OR ───────────────                         │
│                                                             │
│  ○ Create new speaker:                                      │
│    Name: [________________________]                         │
│    Email: [______________________] (optional)               │
│                                                             │
│  ☐ Apply to all 6 segments from this speaker                │
│  ☐ Remember for future recordings                           │
│                                                             │
│                              [Cancel]  [Apply Label]        │
└─────────────────────────────────────────────────────────────┘
```

### 6.2 UI Component Specifications

#### 6.2.1 Segment Component
```tsx
interface SegmentProps {
  segment: Segment;
  speaker?: Speaker;
  isPlaying: boolean;
  onPlayClick: () => void;
  onTextEdit: (text: string) => void;
  onSpeakerClick: () => void;
  onVerifyClick: () => void;
}

// Visual States:
// - Verified speaker: solid color bar, checkmark icon
// - Suggested speaker: dashed border, suggestion tooltip
// - Unknown speaker: orange warning, "Label" button
// - Currently playing: highlighted background, pulse animation
```

#### 6.2.2 Speaker Badge Component
```tsx
interface SpeakerBadgeProps {
  speaker: Speaker;
  size: 'sm' | 'md' | 'lg';
  showConfidence?: boolean;
  isVerified?: boolean;
  onClick?: () => void;
}

// Colors assigned from palette:
const SPEAKER_COLORS = [
  '#3B82F6', // blue
  '#10B981', // green
  '#F59E0B', // amber
  '#EF4444', // red
  '#8B5CF6', // purple
  '#EC4899', // pink
  '#06B6D4', // cyan
  '#F97316', // orange
];
```

---

## 7. Technology Stack

### 7.1 Core Technologies

| Layer | Technology | Rationale |
|-------|------------|-----------|
| **Mac App Framework** | Tauri 2.0 (Rust + WebView) | Native performance, small bundle, macOS APIs access |
| **Frontend** | Next.js 15 + React 19 | Consistent with existing Agentok codebase |
| **UI Components** | Radix UI + TailwindCSS | Consistent with existing design system |
| **State Management** | Zustand | Lightweight, already in use |
| **Backend** | FastAPI (Python 3.11+) | AI/ML ecosystem, existing codebase |
| **Local Database** | SQLite + sqlite-vss | Fast, embedded, vector search |
| **Cloud Sync** | Supabase (optional) | Already integrated, pgvector support |

### 7.2 AI/ML Stack

| Component | Technology | Notes |
|-----------|------------|-------|
| **Speech-to-Text** | OpenAI Whisper (large-v3) | Best accuracy, can run locally or via API |
| **Local Whisper** | faster-whisper | 4x faster than original, INT8 quantization |
| **Speaker Diarization** | pyannote.audio 3.1 | State-of-the-art, Hugging Face integration |
| **Voice Embeddings** | ECAPA-TDNN (SpeechBrain) | 192-dim embeddings, excellent for speaker ID |
| **Alternative Embeddings** | resemblyzer | Lighter weight option |
| **VAD** | Silero VAD | Fast, accurate voice activity detection |

### 7.3 Mac-Specific Technologies

| Feature | Technology | macOS Version |
|---------|------------|---------------|
| **Screen Capture** | ScreenCaptureKit (SCStream) | macOS 13+ |
| **System Audio** | SCStreamAudioOutput | macOS 13+ |
| **Microphone** | AVCaptureSession | macOS 10.7+ |
| **Permissions** | TCC (Privacy Framework) | macOS 10.14+ |
| **Menu Bar App** | Tauri system tray | All |

### 7.4 Python Dependencies

```toml
[project]
name = "scribe-transcription"
version = "1.0.0"
requires-python = ">=3.11"

[project.dependencies]
# Web Framework
fastapi = "^0.115.0"
uvicorn = "^0.32.0"
websockets = "^12.0"

# AI/ML - Transcription
openai-whisper = "^20231117"
faster-whisper = "^1.0.0"

# AI/ML - Speaker Diarization
pyannote-audio = "^3.1.0"
speechbrain = "^1.0.0"

# Audio Processing
librosa = "^0.10.0"
soundfile = "^0.12.0"
pydub = "^0.25.0"

# Vector Search
numpy = "^1.26.0"
sqlite-vss = "^0.1.0"  # or hnswlib for faster search

# Utilities
pydantic = "^2.10.0"
python-multipart = "^0.0.9"
aiofiles = "^23.0.0"
```

---

## 8. Speaker Learning Algorithm

### 8.1 Embedding Extraction Pipeline

```python
from speechbrain.pretrained import EncoderClassifier
import torch
import numpy as np

class SpeakerEmbeddingExtractor:
    """Extract voice embeddings using ECAPA-TDNN model."""

    def __init__(self, device: str = "mps"):  # Use Metal on Mac
        self.model = EncoderClassifier.from_hparams(
            source="speechbrain/spkrec-ecapa-voxceleb",
            savedir="models/ecapa",
            run_opts={"device": device}
        )
        self.embedding_dim = 192

    def extract(self, audio: np.ndarray, sample_rate: int = 16000) -> np.ndarray:
        """Extract 192-dim embedding from audio segment."""
        # Ensure correct format
        if len(audio.shape) > 1:
            audio = audio.mean(axis=1)  # Convert to mono

        # Normalize
        audio = audio / (np.abs(audio).max() + 1e-8)

        # Convert to tensor
        waveform = torch.tensor(audio).unsqueeze(0).float()

        # Extract embedding
        embedding = self.model.encode_batch(waveform)
        return embedding.squeeze().cpu().numpy()

    def compute_similarity(self, emb1: np.ndarray, emb2: np.ndarray) -> float:
        """Compute cosine similarity between two embeddings."""
        return np.dot(emb1, emb2) / (np.linalg.norm(emb1) * np.linalg.norm(emb2))
```

### 8.2 Speaker Profile Manager

```python
from dataclasses import dataclass, field
from typing import List, Optional, Tuple
import numpy as np
from datetime import datetime

@dataclass
class SpeakerProfile:
    id: str
    name: str
    centroid: np.ndarray  # 192-dim average embedding
    samples: List[np.ndarray] = field(default_factory=list)
    sample_count: int = 0
    confidence: float = 0.0
    created_at: datetime = field(default_factory=datetime.utcnow)
    last_updated: datetime = field(default_factory=datetime.utcnow)

    MAX_SAMPLES = 50
    ADAPTATION_RATE = 0.95  # Slow adaptation to prevent drift

    def add_sample(self, embedding: np.ndarray, quality_score: float = 1.0):
        """Add new embedding sample and update centroid."""
        # Add to samples
        self.samples.append(embedding)
        self.sample_count += 1

        # Prune if too many samples (keep diverse samples)
        if len(self.samples) > self.MAX_SAMPLES:
            self._prune_samples()

        # Update centroid with exponential moving average
        if self.centroid is None:
            self.centroid = embedding
        else:
            self.centroid = (
                self.ADAPTATION_RATE * self.centroid +
                (1 - self.ADAPTATION_RATE) * embedding
            )

        # Normalize centroid
        self.centroid = self.centroid / np.linalg.norm(self.centroid)

        # Update confidence based on sample count
        self.confidence = min(1.0, self.sample_count / 20)  # Full confidence at 20 samples
        self.last_updated = datetime.utcnow()

    def _prune_samples(self):
        """Remove samples that are too similar to others (keep diversity)."""
        if len(self.samples) <= self.MAX_SAMPLES:
            return

        # Compute all pairwise similarities
        n = len(self.samples)
        similarities = np.zeros((n, n))
        for i in range(n):
            for j in range(i+1, n):
                sim = np.dot(self.samples[i], self.samples[j])
                similarities[i, j] = similarities[j, i] = sim

        # Remove samples that are most similar to others
        while len(self.samples) > self.MAX_SAMPLES:
            # Find most redundant sample (highest avg similarity to others)
            avg_sims = similarities.mean(axis=1)
            redundant_idx = avg_sims.argmax()

            # Remove
            self.samples.pop(redundant_idx)
            similarities = np.delete(np.delete(similarities, redundant_idx, 0), redundant_idx, 1)


class SpeakerMatcher:
    """Match unknown speakers against known profiles."""

    HIGH_CONFIDENCE_THRESHOLD = 0.75
    SUGGESTION_THRESHOLD = 0.60

    def __init__(self, profiles: List[SpeakerProfile]):
        self.profiles = {p.id: p for p in profiles}

    def find_matches(
        self,
        embedding: np.ndarray
    ) -> List[Tuple[str, str, float, str]]:
        """
        Find matching speakers for an unknown embedding.

        Returns: List of (speaker_id, speaker_name, similarity, confidence_level)
        """
        matches = []

        for profile in self.profiles.values():
            similarity = np.dot(embedding, profile.centroid)

            if similarity >= self.SUGGESTION_THRESHOLD:
                if similarity >= self.HIGH_CONFIDENCE_THRESHOLD:
                    confidence = "high"
                else:
                    confidence = "medium"

                matches.append((
                    profile.id,
                    profile.name,
                    float(similarity),
                    confidence
                ))

        # Sort by similarity descending
        matches.sort(key=lambda x: x[2], reverse=True)
        return matches

    def identify_speaker(
        self,
        embedding: np.ndarray
    ) -> Optional[Tuple[str, float]]:
        """
        Automatically identify speaker if confidence is high enough.

        Returns: (speaker_id, similarity) if confident, None otherwise
        """
        matches = self.find_matches(embedding)

        if matches and matches[0][3] == "high":
            return (matches[0][0], matches[0][2])

        return None
```

### 8.3 Learning Workflow

```python
class SpeakerLearningService:
    """Orchestrates the speaker learning workflow."""

    def __init__(self, db: Database, embedder: SpeakerEmbeddingExtractor):
        self.db = db
        self.embedder = embedder

    async def process_labeling(
        self,
        segment_ids: List[str],
        temp_speaker_id: str,
        speaker_id: Optional[str] = None,
        new_speaker_name: Optional[str] = None
    ) -> Speaker:
        """
        Process user's speaker labeling action.

        1. Create or retrieve speaker profile
        2. Extract embeddings from all labeled segments
        3. Update speaker profile with new samples
        4. Update all segments with speaker_id
        5. Trigger re-matching for other unidentified segments
        """
        # Get or create speaker
        if speaker_id:
            speaker = await self.db.get_speaker(speaker_id)
            profile = await self._load_profile(speaker)
        else:
            speaker = await self.db.create_speaker(name=new_speaker_name)
            profile = SpeakerProfile(
                id=speaker.id,
                name=speaker.name,
                centroid=None,
                samples=[]
            )

        # Get all segments
        segments = await self.db.get_segments(segment_ids)

        # Extract and add embeddings
        for segment in segments:
            audio = await self._load_segment_audio(segment)
            embedding = self.embedder.extract(audio)

            # Add to profile
            profile.add_sample(embedding)

            # Store embedding with segment
            await self.db.update_segment(
                segment.id,
                speaker_id=speaker.id,
                embedding=embedding,
                is_user_verified=True,
                speaker_confidence=1.0
            )

        # Save updated profile
        await self._save_profile(profile)

        # Update speaker stats
        await self.db.update_speaker(
            speaker.id,
            sample_count=profile.sample_count,
            confidence=profile.confidence,
            last_seen_at=datetime.utcnow()
        )

        # Trigger re-matching for other unidentified segments
        await self._rematch_unidentified_segments(segment.recording_id)

        return speaker

    async def _rematch_unidentified_segments(self, recording_id: str):
        """Re-run speaker matching for unidentified segments."""
        # Get all profiles
        profiles = await self._load_all_profiles()
        matcher = SpeakerMatcher(profiles)

        # Get unidentified segments
        segments = await self.db.get_segments_by_recording(
            recording_id,
            filter_unidentified=True
        )

        for segment in segments:
            if segment.embedding is None:
                continue

            match = matcher.identify_speaker(segment.embedding)
            if match:
                speaker_id, similarity = match
                await self.db.update_segment(
                    segment.id,
                    speaker_id=speaker_id,
                    speaker_confidence=similarity,
                    is_user_verified=False  # Auto-matched, not verified
                )
```

---

## 9. Implementation Phases

### Phase 1: Core Foundation (MVP)
**Goal:** Basic transcription from audio files with manual speaker labeling

**Features:**
- [ ] Import audio/video files (MP3, WAV, M4A, MP4, WebM)
- [ ] Whisper transcription (via OpenAI API initially)
- [ ] Speaker diarization (pyannote.audio)
- [ ] Display transcript with temporary speaker labels (Speaker A, B, C)
- [ ] Manual speaker labeling UI
- [ ] Basic speaker profile storage
- [ ] Export to TXT and SRT

**Technical Deliverables:**
- FastAPI backend with transcription pipeline
- SQLite database schema
- Next.js frontend with transcript viewer
- Basic speaker labeling modal

### Phase 2: Speaker Learning
**Goal:** Learn speaker identities from labels and auto-identify in new recordings

**Features:**
- [ ] Voice embedding extraction (ECAPA-TDNN)
- [ ] Speaker profile with embedding storage
- [ ] Automatic speaker matching in new recordings
- [ ] Confidence scores and suggestions
- [ ] Speaker merge functionality
- [ ] Profile management (edit, delete speakers)

**Technical Deliverables:**
- SpeechBrain integration for embeddings
- Vector similarity search (sqlite-vss)
- Speaker matching service
- Incremental learning on corrections

### Phase 3: Mac Screen Capture
**Goal:** Native Mac app with screen/audio capture

**Features:**
- [ ] Tauri app wrapper
- [ ] Screen audio capture (ScreenCaptureKit)
- [ ] Microphone capture
- [ ] Real-time transcription stream
- [ ] Menu bar controls
- [ ] Capture permission handling
- [ ] Local Whisper option (faster-whisper)

**Technical Deliverables:**
- Rust/Tauri native layer
- ScreenCaptureKit bindings
- WebSocket real-time updates
- Local ML inference setup

### Phase 4: Polish & Advanced Features
**Goal:** Production-ready with advanced features

**Features:**
- [ ] Full-text search across all transcripts
- [ ] Export to DOCX with formatting
- [ ] Real-time collaborative viewing
- [ ] Summary generation (LLM-powered)
- [ ] Cloud sync (optional Supabase)
- [ ] Keyboard shortcuts
- [ ] Transcript editing with re-alignment

**Technical Deliverables:**
- FTS5 search integration
- WebSocket collaboration
- LLM summarization pipeline
- Cross-device sync

---

## 10. Success Metrics

### 10.1 Technical Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Transcription accuracy (WER) | < 10% | Word Error Rate on test set |
| Speaker diarization (DER) | < 15% | Diarization Error Rate |
| Speaker ID accuracy | > 90% | After 5+ labeled samples |
| Processing speed | < 0.5x realtime | Time to transcribe / audio duration |
| App launch time | < 2s | Cold start to ready |
| Memory usage | < 500MB | During active transcription |

### 10.2 User Experience Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Time to first transcript | < 3 min | From file upload to viewable |
| Speaker labeling time | < 10s | Per speaker in recording |
| Auto-ID accuracy (user perspective) | > 85% | User confirmations vs corrections |
| Search result latency | < 200ms | Query to results |
| Export time | < 5s | For typical 1-hour recording |

---

## 11. Security & Privacy Considerations

### 11.1 Data Handling

- **Local-first:** All data stored locally by default
- **Voice embeddings:** Non-reversible (cannot reconstruct audio)
- **No cloud by default:** Cloud sync is opt-in
- **Encrypted storage:** SQLite with SQLCipher for sensitive data
- **Secure deletion:** Proper file shredding for deleted recordings

### 11.2 Permissions

- **Screen Recording:** Required for screen capture
- **Microphone:** Required for mic input
- **Accessibility:** Not required
- **Full Disk Access:** Not required

### 11.3 Privacy Policy Requirements

- Clear disclosure of audio processing
- Voice embedding storage explanation
- Cloud sync data handling (if enabled)
- Data retention and deletion policies

---

## 12. Open Questions & Decisions Needed

### 12.1 Architecture Decisions

| Question | Options | Recommendation |
|----------|---------|----------------|
| Whisper deployment | API vs Local | Start with API, add local later |
| App framework | Electron vs Tauri | Tauri (smaller, faster, Rust) |
| Vector DB | sqlite-vss vs hnswlib vs pgvector | sqlite-vss (embedded, simple) |
| State sync | REST polling vs WebSocket | WebSocket for real-time |

### 12.2 Product Decisions

| Question | Options | Needs Input |
|----------|---------|-------------|
| Pricing model | Free, Freemium, Paid | Business decision |
| Cloud sync | Required or Optional | Privacy vs convenience |
| Team features | MVP or Later | Scope decision |
| Mobile app | Phase 2 or Later | Platform priority |

### 12.3 Technical Risks

| Risk | Mitigation |
|------|------------|
| Whisper API costs at scale | Local inference option |
| Speaker ID accuracy with few samples | Explicit confidence indicators |
| macOS permission UX friction | Clear onboarding flow |
| Large audio file processing time | Chunked processing, progress UI |

---

## 13. Appendix

### A. Glossary

- **Diarization:** Segmenting audio by speaker ("who spoke when")
- **WER:** Word Error Rate - transcription accuracy metric
- **DER:** Diarization Error Rate - speaker segmentation accuracy
- **ECAPA-TDNN:** Voice embedding neural network architecture
- **Embedding:** Fixed-size vector representing voice characteristics
- **Centroid:** Average embedding representing a speaker's voice
- **VAD:** Voice Activity Detection - finding speech vs silence

### B. Reference Implementations

- [pyannote.audio](https://github.com/pyannote/pyannote-audio) - Speaker diarization
- [faster-whisper](https://github.com/guillaumekln/faster-whisper) - Optimized Whisper
- [SpeechBrain](https://github.com/speechbrain/speechbrain) - Voice embeddings
- [Tauri](https://tauri.app/) - Desktop app framework

### C. Related Research Papers

1. "Whisper: Robust Speech Recognition via Large-Scale Weak Supervision" (Radford et al., 2022)
2. "ECAPA-TDNN: Emphasized Channel Attention for Speaker Verification" (Desplanques et al., 2020)
3. "pyannote.audio: neural building blocks for speaker diarization" (Bredin et al., 2020)

---

*End of Specification*
