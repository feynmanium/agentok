"""Scribe data models."""

from .database import (
    Recording,
    Speaker,
    SpeakerEmbedding,
    Segment,
    Base,
    get_db,
    init_db,
)
from .schemas import (
    RecordingCreate,
    RecordingResponse,
    RecordingWithSegments,
    SpeakerCreate,
    SpeakerResponse,
    SpeakerUpdate,
    SegmentCreate,
    SegmentResponse,
    SegmentUpdate,
    SpeakerLabelRequest,
    SpeakerMatchResult,
    TranscriptionConfig,
    TranscriptionProgress,
    ExportRequest,
    ExportFormat,
)

__all__ = [
    # Database models
    "Recording",
    "Speaker",
    "SpeakerEmbedding",
    "Segment",
    "Base",
    "get_db",
    "init_db",
    # Schemas
    "RecordingCreate",
    "RecordingResponse",
    "RecordingWithSegments",
    "SpeakerCreate",
    "SpeakerResponse",
    "SpeakerUpdate",
    "SegmentCreate",
    "SegmentResponse",
    "SegmentUpdate",
    "SpeakerLabelRequest",
    "SpeakerMatchResult",
    "TranscriptionConfig",
    "TranscriptionProgress",
    "ExportRequest",
    "ExportFormat",
]
