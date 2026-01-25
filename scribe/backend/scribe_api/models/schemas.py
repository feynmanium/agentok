"""Pydantic schemas for Scribe API."""

from datetime import datetime
from enum import Enum
from typing import Optional, List, Literal

from pydantic import BaseModel, Field


# ============================================
# Enums
# ============================================

class SourceType(str, Enum):
    SCREEN_CAPTURE = "screen_capture"
    AUDIO_FILE = "audio_file"
    LIVE_MIC = "live_mic"


class RecordingStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class ExportFormat(str, Enum):
    TXT = "txt"
    SRT = "srt"
    VTT = "vtt"
    JSON = "json"


# ============================================
# Recording Schemas
# ============================================

class RecordingCreate(BaseModel):
    title: str = Field(..., min_length=1, max_length=255)
    description: Optional[str] = None
    source_type: SourceType = SourceType.AUDIO_FILE


class RecordingResponse(BaseModel):
    id: str
    title: str
    description: Optional[str] = None
    source_type: SourceType
    source_path: Optional[str] = None
    duration_ms: Optional[int] = None
    status: RecordingStatus
    audio_format: Optional[str] = None
    created_at: datetime
    updated_at: datetime

    class Config:
        from_attributes = True


class SpeakerSummary(BaseModel):
    """Summary of a speaker's participation in a recording."""
    speaker_id: Optional[str] = None
    speaker_name: str
    temp_speaker_id: Optional[str] = None
    color: Optional[str] = None
    total_duration_ms: int
    segment_count: int
    word_count: int
    percentage: float


class RecordingWithSegments(RecordingResponse):
    segments: List["SegmentResponse"] = []
    speaker_summary: List[SpeakerSummary] = []


# ============================================
# Speaker Schemas
# ============================================

class SpeakerCreate(BaseModel):
    name: str = Field(..., min_length=1, max_length=255)
    email: Optional[str] = None
    color: Optional[str] = None


class SpeakerUpdate(BaseModel):
    name: Optional[str] = Field(None, min_length=1, max_length=255)
    email: Optional[str] = None
    color: Optional[str] = None


class SpeakerResponse(BaseModel):
    id: str
    name: str
    email: Optional[str] = None
    color: Optional[str] = None
    sample_count: int = 0
    confidence: float = 0.0
    first_seen_at: Optional[datetime] = None
    last_seen_at: Optional[datetime] = None
    created_at: datetime

    class Config:
        from_attributes = True


class SpeakerMatchResult(BaseModel):
    """Result of matching an unknown speaker against known profiles."""
    speaker_id: str
    speaker_name: str
    similarity: float
    confidence: Literal["high", "medium", "low"]


# ============================================
# Segment Schemas
# ============================================

class WordTimestamp(BaseModel):
    word: str
    start_ms: int
    end_ms: int
    confidence: Optional[float] = None


class SegmentCreate(BaseModel):
    recording_id: str
    text: str
    start_ms: int
    end_ms: int
    temp_speaker_id: Optional[str] = None
    confidence: Optional[float] = None
    word_timestamps: Optional[List[WordTimestamp]] = None


class SegmentResponse(BaseModel):
    id: str
    recording_id: str
    speaker_id: Optional[str] = None
    temp_speaker_id: Optional[str] = None
    text: str
    start_ms: int
    end_ms: int
    confidence: Optional[float] = None
    speaker_confidence: Optional[float] = None
    is_user_verified: bool = False
    has_overlap: bool = False  # True if overlapping speech detected
    word_timestamps: Optional[List[WordTimestamp]] = None
    speaker: Optional[SpeakerResponse] = None
    created_at: datetime

    class Config:
        from_attributes = True


class SegmentUpdate(BaseModel):
    text: Optional[str] = None
    speaker_id: Optional[str] = None
    is_user_verified: Optional[bool] = None


# ============================================
# Speaker Labeling Schemas
# ============================================

class SpeakerLabelRequest(BaseModel):
    """Request to label segments with a speaker identity."""
    segment_ids: List[str] = Field(..., min_length=1)
    temp_speaker_id: str
    speaker_id: Optional[str] = None  # Existing speaker ID
    new_speaker_name: Optional[str] = None  # Create new speaker
    apply_to_recording: bool = True  # Apply to all segments with this temp_speaker_id


class SpeakerLabelResponse(BaseModel):
    updated_count: int
    speaker: SpeakerResponse


# ============================================
# Transcription Schemas
# ============================================

class TranscriptionConfig(BaseModel):
    """Configuration for transcription processing."""
    language: str = "auto"
    model: str = "whisper-1"  # OpenAI model or local model path
    enable_diarization: bool = True
    enable_speaker_matching: bool = True
    speaker_matching_threshold: float = 0.75
    speaker_suggestion_threshold: float = 0.60
    enable_word_timestamps: bool = True


class TranscriptionProgress(BaseModel):
    """Progress update during transcription."""
    recording_id: str
    stage: Literal["uploading", "processing", "diarizing", "transcribing", "identifying", "complete", "failed"]
    progress: float = 0.0  # 0-100
    message: Optional[str] = None
    current_segment: Optional[int] = None
    total_segments: Optional[int] = None


# ============================================
# Export Schemas
# ============================================

class ExportRequest(BaseModel):
    format: ExportFormat = ExportFormat.TXT
    include_timestamps: bool = True
    include_speaker_names: bool = True


# ============================================
# Search Schemas
# ============================================

class SearchQuery(BaseModel):
    q: str = Field(..., min_length=1)
    speaker_id: Optional[str] = None
    recording_id: Optional[str] = None
    date_from: Optional[datetime] = None
    date_to: Optional[datetime] = None
    limit: int = Field(50, ge=1, le=100)
    offset: int = Field(0, ge=0)


class SearchResult(BaseModel):
    segment: SegmentResponse
    recording: RecordingResponse
    highlight: str


class SearchResponse(BaseModel):
    results: List[SearchResult]
    total: int


# Update forward references
RecordingWithSegments.model_rebuild()
