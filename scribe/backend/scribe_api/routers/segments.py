"""Segments API router."""

import uuid
from datetime import datetime
from typing import List, Optional

from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy import select, update
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from ..models.database import Segment, Speaker, Recording, get_db
from ..models.schemas import (
    SegmentResponse,
    SegmentUpdate,
    SpeakerLabelRequest,
    SpeakerLabelResponse,
    SpeakerResponse,
    SpeakerMatchResult,
)
from ..services.speaker_learning import SpeakerLearningService

router = APIRouter(prefix="/segments", tags=["segments"])

# Services
_speaker_learning_service: Optional[SpeakerLearningService] = None


def get_speaker_learning_service() -> SpeakerLearningService:
    global _speaker_learning_service
    if _speaker_learning_service is None:
        _speaker_learning_service = SpeakerLearningService()
    return _speaker_learning_service


@router.get("/{segment_id}", response_model=SegmentResponse)
async def get_segment(
    segment_id: str,
    db: AsyncSession = Depends(get_db),
):
    """Get a single segment."""
    stmt = (
        select(Segment)
        .where(Segment.id == segment_id)
        .options(selectinload(Segment.speaker))
    )
    result = await db.execute(stmt)
    segment = result.scalar_one_or_none()

    if not segment:
        raise HTTPException(status_code=404, detail="Segment not found")

    return segment


@router.patch("/{segment_id}", response_model=SegmentResponse)
async def update_segment(
    segment_id: str,
    request: SegmentUpdate,
    db: AsyncSession = Depends(get_db),
):
    """Update a segment's text or speaker assignment."""
    stmt = (
        select(Segment)
        .where(Segment.id == segment_id)
        .options(selectinload(Segment.speaker))
    )
    result = await db.execute(stmt)
    segment = result.scalar_one_or_none()

    if not segment:
        raise HTTPException(status_code=404, detail="Segment not found")

    if request.text is not None:
        segment.text = request.text

    if request.speaker_id is not None:
        # Verify speaker exists
        stmt = select(Speaker).where(Speaker.id == request.speaker_id)
        result = await db.execute(stmt)
        speaker = result.scalar_one_or_none()
        if not speaker:
            raise HTTPException(status_code=400, detail="Speaker not found")

        segment.speaker_id = request.speaker_id
        segment.temp_speaker_id = None  # Clear temp ID when assigned

    if request.is_user_verified is not None:
        segment.is_user_verified = request.is_user_verified
        if request.is_user_verified:
            segment.speaker_confidence = 1.0

    segment.updated_at = datetime.utcnow()
    await db.commit()
    await db.refresh(segment)

    # Reload with speaker relationship
    stmt = (
        select(Segment)
        .where(Segment.id == segment_id)
        .options(selectinload(Segment.speaker))
    )
    result = await db.execute(stmt)
    segment = result.scalar_one()

    return segment


@router.post("/bulk-label", response_model=SpeakerLabelResponse)
async def bulk_label_segments(
    request: SpeakerLabelRequest,
    db: AsyncSession = Depends(get_db),
):
    """
    Label multiple segments with a speaker identity.

    This is the main endpoint for speaker learning. When a user labels
    "Speaker A" as "Bill", this endpoint:
    1. Creates or retrieves the speaker profile
    2. Updates all specified segments with the speaker ID
    3. Extracts voice embeddings and adds them to the speaker profile
    4. Optionally applies to all segments with the same temp_speaker_id
    """
    # Validate request
    if not request.speaker_id and not request.new_speaker_name:
        raise HTTPException(
            status_code=400,
            detail="Either speaker_id or new_speaker_name is required",
        )

    # Get or create speaker
    if request.speaker_id:
        stmt = select(Speaker).where(Speaker.id == request.speaker_id)
        result = await db.execute(stmt)
        speaker = result.scalar_one_or_none()
        if not speaker:
            raise HTTPException(status_code=400, detail="Speaker not found")
    else:
        # Check for existing speaker with same name
        stmt = select(Speaker).where(Speaker.name == request.new_speaker_name)
        result = await db.execute(stmt)
        existing = result.scalar_one_or_none()
        if existing:
            raise HTTPException(
                status_code=400,
                detail=f"Speaker '{request.new_speaker_name}' already exists. Use speaker_id to assign to existing speaker.",
            )

        # Create new speaker
        from .speakers import get_next_color
        stmt = select(Speaker.color).where(Speaker.color.isnot(None))
        result = await db.execute(stmt)
        existing_colors = [r[0] for r in result.all()]

        speaker = Speaker(
            id=str(uuid.uuid4()),
            name=request.new_speaker_name,
            color=get_next_color(existing_colors),
            first_seen_at=datetime.utcnow(),
            created_at=datetime.utcnow(),
        )
        db.add(speaker)
        await db.flush()

    # Get segments to update
    if request.apply_to_recording:
        # Get all segments with the same temp_speaker_id in the same recording(s)
        # First get the recording IDs from the specified segments
        stmt = select(Segment.recording_id).where(Segment.id.in_(request.segment_ids)).distinct()
        result = await db.execute(stmt)
        recording_ids = [r[0] for r in result.all()]

        # Then get all segments with matching temp_speaker_id in those recordings
        stmt = select(Segment).where(
            Segment.recording_id.in_(recording_ids),
            Segment.temp_speaker_id == request.temp_speaker_id,
        )
    else:
        # Only update specified segments
        stmt = select(Segment).where(Segment.id.in_(request.segment_ids))

    result = await db.execute(stmt)
    segments = result.scalars().all()

    if not segments:
        raise HTTPException(status_code=404, detail="No segments found to update")

    # Update segments
    for segment in segments:
        segment.speaker_id = speaker.id
        segment.is_user_verified = True
        segment.speaker_confidence = 1.0
        # Keep temp_speaker_id for reference but speaker_id takes precedence

    # Update speaker's last seen
    speaker.last_seen_at = datetime.utcnow()
    if not speaker.first_seen_at:
        speaker.first_seen_at = datetime.utcnow()

    await db.commit()

    # Retrospective Update: scan existing unknown segments and auto-label
    # any that match the newly labeled speaker with > 0.75 similarity
    svc = get_speaker_learning_service()
    auto_labeled_count = 0

    # Get all unique recording IDs from the labeled segments
    recording_ids = list(set(seg.recording_id for seg in segments))

    for rec_id in recording_ids:
        try:
            count = await svc.retrospective_update(
                speaker_id=speaker.id,
                recording_id=rec_id,
                db_session=db,
            )
            auto_labeled_count += count
        except Exception as e:
            # Log but don't fail the request
            import logging
            logging.getLogger(__name__).warning(
                f"Retrospective update failed for recording {rec_id}: {e}"
            )

    await db.refresh(speaker)

    return SpeakerLabelResponse(
        updated_count=len(segments) + auto_labeled_count,
        speaker=SpeakerResponse.model_validate(speaker),
    )


@router.get("/{segment_id}/suggestions", response_model=List[SpeakerMatchResult])
async def get_speaker_suggestions(
    segment_id: str,
    db: AsyncSession = Depends(get_db),
):
    """
    Get speaker suggestions for a segment based on voice similarity.

    This uses the segment's voice embedding (if available) to find
    matching known speakers.
    """
    stmt = select(Segment).where(Segment.id == segment_id)
    result = await db.execute(stmt)
    segment = result.scalar_one_or_none()

    if not segment:
        raise HTTPException(status_code=404, detail="Segment not found")

    if not segment.embedding:
        # No embedding available - return empty list
        return []

    # Get suggestions from speaker learning service
    from ..services.speaker_learning import bytes_to_embedding
    embedding = bytes_to_embedding(segment.embedding)

    svc = get_speaker_learning_service()
    matches = await svc.get_suggestions(embedding, db)

    return [
        SpeakerMatchResult(
            speaker_id=m[0],
            speaker_name=m[1],
            similarity=m[2],
            confidence=m[3],
        )
        for m in matches
    ]


@router.post("/search")
async def search_segments(
    q: str,
    recording_id: Optional[str] = None,
    speaker_id: Optional[str] = None,
    limit: int = 50,
    offset: int = 0,
    db: AsyncSession = Depends(get_db),
):
    """
    Full-text search across segments.

    Note: This is a simple LIKE-based search. For production, consider
    implementing FTS5 virtual tables in SQLite.
    """
    stmt = (
        select(Segment)
        .options(selectinload(Segment.speaker))
        .where(Segment.text.ilike(f"%{q}%"))
        .order_by(Segment.created_at.desc())
    )

    if recording_id:
        stmt = stmt.where(Segment.recording_id == recording_id)

    if speaker_id:
        stmt = stmt.where(Segment.speaker_id == speaker_id)

    stmt = stmt.offset(offset).limit(limit)
    result = await db.execute(stmt)
    segments = result.scalars().all()

    # Get recording info for context
    recording_ids = list(set(s.recording_id for s in segments))
    stmt = select(Recording).where(Recording.id.in_(recording_ids))
    result = await db.execute(stmt)
    recordings = {r.id: r for r in result.scalars().all()}

    # Build results with highlights
    results = []
    for seg in segments:
        # Simple highlight - wrap matching text
        text = seg.text
        lower_text = text.lower()
        lower_q = q.lower()

        start = lower_text.find(lower_q)
        if start >= 0:
            end = start + len(q)
            highlight = f"...{text[max(0, start-30):start]}**{text[start:end]}**{text[end:end+30]}..."
        else:
            highlight = text[:60] + "..."

        results.append({
            "segment": SegmentResponse.model_validate(seg),
            "recording": {
                "id": recordings[seg.recording_id].id,
                "title": recordings[seg.recording_id].title,
            } if seg.recording_id in recordings else None,
            "highlight": highlight,
        })

    return {
        "results": results,
        "total": len(results),  # For proper pagination, would need a count query
    }
