"""SQLAlchemy database models for Scribe."""

import os
from datetime import datetime
from enum import Enum
from typing import AsyncGenerator, Optional
from contextlib import asynccontextmanager

from sqlalchemy import (
    Column,
    String,
    Integer,
    Float,
    Boolean,
    DateTime,
    Text,
    LargeBinary,
    ForeignKey,
    JSON,
    create_engine,
    event,
)
from sqlalchemy.ext.asyncio import AsyncSession, create_async_engine, async_sessionmaker
from sqlalchemy.orm import DeclarativeBase, relationship


class SourceType(str, Enum):
    SCREEN_CAPTURE = "screen_capture"
    AUDIO_FILE = "audio_file"
    LIVE_MIC = "live_mic"


class RecordingStatus(str, Enum):
    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"


class Base(DeclarativeBase):
    """Base class for all models."""
    pass


class Recording(Base):
    """Recording session - one per screen recording or audio file."""

    __tablename__ = "recordings"

    id = Column(String(36), primary_key=True)
    title = Column(String(255), nullable=False)
    description = Column(Text, nullable=True)
    source_type = Column(String(50), nullable=False, default=SourceType.AUDIO_FILE.value)
    source_path = Column(Text, nullable=True)
    duration_ms = Column(Integer, nullable=True)
    status = Column(String(50), nullable=False, default=RecordingStatus.PENDING.value)
    audio_format = Column(String(20), nullable=True)
    sample_rate = Column(Integer, default=16000)
    error_message = Column(Text, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    metadata = Column(JSON, nullable=True)

    # Relationships
    segments = relationship("Segment", back_populates="recording", cascade="all, delete-orphan")


class Speaker(Base):
    """Speaker profile - learned from user labels."""

    __tablename__ = "speakers"

    id = Column(String(36), primary_key=True)
    name = Column(String(255), nullable=False)
    email = Column(String(255), nullable=True)
    avatar_path = Column(Text, nullable=True)
    color = Column(String(7), nullable=True)  # Hex color for UI
    embedding_centroid = Column(LargeBinary, nullable=True)  # 192-dim float32 vector
    sample_count = Column(Integer, default=0)
    confidence = Column(Float, default=0.0)
    first_seen_at = Column(DateTime, nullable=True)
    last_seen_at = Column(DateTime, nullable=True)
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)
    metadata = Column(JSON, nullable=True)

    # Relationships
    embeddings = relationship("SpeakerEmbedding", back_populates="speaker", cascade="all, delete-orphan")
    segments = relationship("Segment", back_populates="speaker")


class SpeakerEmbedding(Base):
    """Voice embedding samples for each speaker (for retraining)."""

    __tablename__ = "speaker_embeddings"

    id = Column(String(36), primary_key=True)
    speaker_id = Column(String(36), ForeignKey("speakers.id", ondelete="CASCADE"), nullable=False)
    embedding = Column(LargeBinary, nullable=False)  # 192-dim float32 vector
    source_segment_id = Column(String(36), ForeignKey("segments.id", ondelete="SET NULL"), nullable=True)
    quality_score = Column(Float, default=1.0)
    created_at = Column(DateTime, default=datetime.utcnow)

    # Relationships
    speaker = relationship("Speaker", back_populates="embeddings")


class Segment(Base):
    """Transcript segment - individual utterance."""

    __tablename__ = "segments"

    id = Column(String(36), primary_key=True)
    recording_id = Column(String(36), ForeignKey("recordings.id", ondelete="CASCADE"), nullable=False)
    speaker_id = Column(String(36), ForeignKey("speakers.id", ondelete="SET NULL"), nullable=True)
    temp_speaker_id = Column(String(50), nullable=True)  # e.g., "Speaker A"
    text = Column(Text, nullable=False)
    start_ms = Column(Integer, nullable=False)
    end_ms = Column(Integer, nullable=False)
    confidence = Column(Float, nullable=True)
    speaker_confidence = Column(Float, nullable=True)
    is_user_verified = Column(Boolean, default=False)  # True if user manually labeled this segment
    has_overlap = Column(Boolean, default=False)  # True if overlapping speech detected (> 500ms)
    embedding = Column(LargeBinary, nullable=True)  # Voice embedding for this segment
    word_timestamps = Column(JSON, nullable=True)  # Array of {word, start_ms, end_ms}
    created_at = Column(DateTime, default=datetime.utcnow)
    updated_at = Column(DateTime, default=datetime.utcnow, onupdate=datetime.utcnow)

    # Relationships
    recording = relationship("Recording", back_populates="segments")
    speaker = relationship("Speaker", back_populates="segments")


# Database setup
DATABASE_URL = os.getenv("DATABASE_URL", "sqlite+aiosqlite:///./data/scribe.db")

# Ensure data directory exists
os.makedirs("data", exist_ok=True)

engine = create_async_engine(
    DATABASE_URL,
    echo=os.getenv("DEBUG", "false").lower() == "true",
)

async_session = async_sessionmaker(
    engine,
    class_=AsyncSession,
    expire_on_commit=False,
)


async def get_db() -> AsyncGenerator[AsyncSession, None]:
    """Dependency for getting database session."""
    async with async_session() as session:
        try:
            yield session
            await session.commit()
        except Exception:
            await session.rollback()
            raise


async def init_db():
    """Initialize database tables."""
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)


# Enable foreign keys and WAL mode for SQLite
# WAL mode provides better concurrency for reads during writes
@event.listens_for(engine.sync_engine, "connect")
def set_sqlite_pragma(dbapi_connection, connection_record):
    cursor = dbapi_connection.cursor()
    cursor.execute("PRAGMA foreign_keys=ON")
    cursor.execute("PRAGMA journal_mode=WAL")  # Better concurrent access
    cursor.execute("PRAGMA busy_timeout=5000")  # 5 second timeout for locks
    cursor.close()
