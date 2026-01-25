"""Speakers API router."""

import uuid
from datetime import datetime
from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession

from ..models.database import Speaker, Segment, get_db
from ..models.schemas import (
    SpeakerCreate,
    SpeakerResponse,
    SpeakerUpdate,
)

router = APIRouter(prefix="/speakers", tags=["speakers"])

# Color palette for auto-assigning speaker colors
SPEAKER_COLORS = [
    "#3B82F6",  # blue
    "#10B981",  # green
    "#F59E0B",  # amber
    "#EF4444",  # red
    "#8B5CF6",  # purple
    "#EC4899",  # pink
    "#06B6D4",  # cyan
    "#F97316",  # orange
    "#84CC16",  # lime
    "#6366F1",  # indigo
]


def get_next_color(existing_colors: List[str]) -> str:
    """Get the next available color from the palette."""
    for color in SPEAKER_COLORS:
        if color not in existing_colors:
            return color
    # If all colors used, cycle back
    return SPEAKER_COLORS[len(existing_colors) % len(SPEAKER_COLORS)]


@router.get("", response_model=List[SpeakerResponse])
async def list_speakers(
    limit: int = 100,
    offset: int = 0,
    db: AsyncSession = Depends(get_db),
):
    """List all known speakers."""
    stmt = (
        select(Speaker)
        .order_by(Speaker.last_seen_at.desc().nullslast(), Speaker.name)
        .offset(offset)
        .limit(limit)
    )
    result = await db.execute(stmt)
    speakers = result.scalars().all()
    return speakers


@router.post("", response_model=SpeakerResponse)
async def create_speaker(
    request: SpeakerCreate,
    db: AsyncSession = Depends(get_db),
):
    """Create a new speaker profile."""
    # Check for duplicate name
    stmt = select(Speaker).where(Speaker.name == request.name)
    result = await db.execute(stmt)
    existing = result.scalar_one_or_none()
    if existing:
        raise HTTPException(
            status_code=400,
            detail=f"Speaker with name '{request.name}' already exists",
        )

    # Get existing colors to assign a new one
    stmt = select(Speaker.color).where(Speaker.color.isnot(None))
    result = await db.execute(stmt)
    existing_colors = [r[0] for r in result.all()]

    color = request.color or get_next_color(existing_colors)

    speaker = Speaker(
        id=str(uuid.uuid4()),
        name=request.name,
        email=request.email,
        color=color,
        created_at=datetime.utcnow(),
    )
    db.add(speaker)
    await db.commit()
    await db.refresh(speaker)

    return speaker


@router.get("/{speaker_id}", response_model=SpeakerResponse)
async def get_speaker(
    speaker_id: str,
    db: AsyncSession = Depends(get_db),
):
    """Get speaker details."""
    stmt = select(Speaker).where(Speaker.id == speaker_id)
    result = await db.execute(stmt)
    speaker = result.scalar_one_or_none()

    if not speaker:
        raise HTTPException(status_code=404, detail="Speaker not found")

    return speaker


@router.patch("/{speaker_id}", response_model=SpeakerResponse)
async def update_speaker(
    speaker_id: str,
    request: SpeakerUpdate,
    db: AsyncSession = Depends(get_db),
):
    """Update speaker profile."""
    stmt = select(Speaker).where(Speaker.id == speaker_id)
    result = await db.execute(stmt)
    speaker = result.scalar_one_or_none()

    if not speaker:
        raise HTTPException(status_code=404, detail="Speaker not found")

    # Check for duplicate name if changing name
    if request.name and request.name != speaker.name:
        stmt = select(Speaker).where(Speaker.name == request.name)
        result = await db.execute(stmt)
        existing = result.scalar_one_or_none()
        if existing:
            raise HTTPException(
                status_code=400,
                detail=f"Speaker with name '{request.name}' already exists",
            )
        speaker.name = request.name

    if request.email is not None:
        speaker.email = request.email

    if request.color is not None:
        speaker.color = request.color

    speaker.updated_at = datetime.utcnow()
    await db.commit()
    await db.refresh(speaker)

    return speaker


@router.delete("/{speaker_id}")
async def delete_speaker(
    speaker_id: str,
    db: AsyncSession = Depends(get_db),
):
    """
    Delete a speaker profile.

    Segments will have their speaker_id set to NULL.
    """
    stmt = select(Speaker).where(Speaker.id == speaker_id)
    result = await db.execute(stmt)
    speaker = result.scalar_one_or_none()

    if not speaker:
        raise HTTPException(status_code=404, detail="Speaker not found")

    await db.delete(speaker)
    await db.commit()

    return {"status": "deleted"}


@router.post("/{speaker_id}/merge")
async def merge_speakers(
    speaker_id: str,
    source_speaker_id: str,
    db: AsyncSession = Depends(get_db),
):
    """
    Merge another speaker into this one.

    All segments from source_speaker will be reassigned to this speaker,
    and voice profiles will be combined.
    """
    # Get both speakers
    stmt = select(Speaker).where(Speaker.id == speaker_id)
    result = await db.execute(stmt)
    target_speaker = result.scalar_one_or_none()

    if not target_speaker:
        raise HTTPException(status_code=404, detail="Target speaker not found")

    stmt = select(Speaker).where(Speaker.id == source_speaker_id)
    result = await db.execute(stmt)
    source_speaker = result.scalar_one_or_none()

    if not source_speaker:
        raise HTTPException(status_code=404, detail="Source speaker not found")

    if speaker_id == source_speaker_id:
        raise HTTPException(status_code=400, detail="Cannot merge speaker with itself")

    # Update all segments from source to target
    stmt = select(Segment).where(Segment.speaker_id == source_speaker_id)
    result = await db.execute(stmt)
    segments = result.scalars().all()

    for segment in segments:
        segment.speaker_id = speaker_id

    # Merge sample counts
    target_speaker.sample_count += source_speaker.sample_count

    # Update confidence (weighted average)
    if target_speaker.sample_count > 0:
        target_weight = target_speaker.sample_count / (target_speaker.sample_count + source_speaker.sample_count)
        source_weight = 1 - target_weight
        target_speaker.confidence = (
            target_speaker.confidence * target_weight +
            source_speaker.confidence * source_weight
        )

    # Update first/last seen
    if source_speaker.first_seen_at:
        if not target_speaker.first_seen_at or source_speaker.first_seen_at < target_speaker.first_seen_at:
            target_speaker.first_seen_at = source_speaker.first_seen_at

    if source_speaker.last_seen_at:
        if not target_speaker.last_seen_at or source_speaker.last_seen_at > target_speaker.last_seen_at:
            target_speaker.last_seen_at = source_speaker.last_seen_at

    # Move embeddings from source to target
    from ..models.database import SpeakerEmbedding

    stmt = select(SpeakerEmbedding).where(SpeakerEmbedding.speaker_id == source_speaker_id)
    result = await db.execute(stmt)
    embeddings = result.scalars().all()

    for emb in embeddings:
        emb.speaker_id = speaker_id

    # Delete source speaker
    await db.delete(source_speaker)
    await db.commit()
    await db.refresh(target_speaker)

    return {
        "status": "merged",
        "merged_segments": len(segments),
        "merged_embeddings": len(embeddings),
        "speaker": SpeakerResponse.model_validate(target_speaker),
    }


@router.get("/{speaker_id}/stats")
async def get_speaker_stats(
    speaker_id: str,
    db: AsyncSession = Depends(get_db),
):
    """Get detailed statistics for a speaker."""
    stmt = select(Speaker).where(Speaker.id == speaker_id)
    result = await db.execute(stmt)
    speaker = result.scalar_one_or_none()

    if not speaker:
        raise HTTPException(status_code=404, detail="Speaker not found")

    # Get segment stats
    stmt = select(
        func.count(Segment.id).label("segment_count"),
        func.sum(Segment.end_ms - Segment.start_ms).label("total_duration_ms"),
        func.count(func.distinct(Segment.recording_id)).label("recording_count"),
    ).where(Segment.speaker_id == speaker_id)

    result = await db.execute(stmt)
    row = result.one()

    # Calculate total words
    stmt = select(Segment.text).where(Segment.speaker_id == speaker_id)
    result = await db.execute(stmt)
    texts = result.scalars().all()
    total_words = sum(len(text.split()) for text in texts)

    return {
        "speaker": SpeakerResponse.model_validate(speaker),
        "stats": {
            "segment_count": row.segment_count or 0,
            "total_duration_ms": row.total_duration_ms or 0,
            "recording_count": row.recording_count or 0,
            "total_words": total_words,
            "sample_count": speaker.sample_count,
            "confidence": speaker.confidence,
        },
    }
