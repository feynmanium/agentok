"""Recordings API router."""

import asyncio
import os
import shutil
import uuid
from pathlib import Path
from typing import List, Optional

from fastapi import APIRouter, Depends, File, Form, HTTPException, UploadFile, BackgroundTasks
from fastapi.responses import Response
from sqlalchemy import select, func
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from ..models.database import Recording, Segment, Speaker, get_db, RecordingStatus
from ..models.schemas import (
    RecordingCreate,
    RecordingResponse,
    RecordingWithSegments,
    SpeakerSummary,
    SegmentResponse,
    TranscriptionConfig,
    ExportRequest,
    ExportFormat,
)
from ..services.transcription import TranscriptionService
from ..services.diarization import DiarizationService
from ..services.speaker_learning import SpeakerLearningService, bytes_to_embedding
from ..services.export import ExportService

router = APIRouter(prefix="/recordings", tags=["recordings"])

# Storage directory for uploaded files
UPLOAD_DIR = Path("data/uploads")
UPLOAD_DIR.mkdir(parents=True, exist_ok=True)

# Services (initialized lazily)
_transcription_service: Optional[TranscriptionService] = None
_diarization_service: Optional[DiarizationService] = None
_speaker_learning_service: Optional[SpeakerLearningService] = None
_export_service = ExportService()


def get_transcription_service() -> TranscriptionService:
    global _transcription_service
    if _transcription_service is None:
        _transcription_service = TranscriptionService()
    return _transcription_service


def get_diarization_service() -> DiarizationService:
    global _diarization_service
    if _diarization_service is None:
        _diarization_service = DiarizationService()
    return _diarization_service


def get_speaker_learning_service() -> SpeakerLearningService:
    global _speaker_learning_service
    if _speaker_learning_service is None:
        _speaker_learning_service = SpeakerLearningService()
    return _speaker_learning_service


@router.get("", response_model=List[RecordingResponse])
async def list_recordings(
    limit: int = 50,
    offset: int = 0,
    status: Optional[str] = None,
    db: AsyncSession = Depends(get_db),
):
    """List all recordings."""
    stmt = select(Recording).order_by(Recording.created_at.desc())

    if status:
        stmt = stmt.where(Recording.status == status)

    stmt = stmt.offset(offset).limit(limit)
    result = await db.execute(stmt)
    recordings = result.scalars().all()

    return recordings


@router.post("", response_model=RecordingResponse)
async def create_recording(
    background_tasks: BackgroundTasks,
    title: str = Form(...),
    description: Optional[str] = Form(None),
    audio_file: UploadFile = File(...),
    enable_diarization: bool = Form(True),
    language: str = Form("auto"),
    db: AsyncSession = Depends(get_db),
):
    """
    Create a new recording by uploading an audio file.

    The transcription will be processed in the background.
    """
    # Validate file type
    allowed_extensions = {".mp3", ".wav", ".m4a", ".mp4", ".webm", ".ogg", ".flac"}
    file_ext = Path(audio_file.filename).suffix.lower()
    if file_ext not in allowed_extensions:
        raise HTTPException(
            status_code=400,
            detail=f"Unsupported file format. Allowed: {', '.join(allowed_extensions)}",
        )

    # Generate unique ID and save file
    recording_id = str(uuid.uuid4())
    file_path = UPLOAD_DIR / f"{recording_id}{file_ext}"

    try:
        with open(file_path, "wb") as f:
            shutil.copyfileobj(audio_file.file, f)
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Failed to save file: {e}")

    # Create recording record
    recording = Recording(
        id=recording_id,
        title=title,
        description=description,
        source_type="audio_file",
        source_path=str(file_path),
        status=RecordingStatus.PENDING.value,
        audio_format=file_ext.lstrip("."),
    )
    db.add(recording)
    await db.commit()
    await db.refresh(recording)

    # Start background transcription
    config = TranscriptionConfig(
        language=language,
        enable_diarization=enable_diarization,
    )
    background_tasks.add_task(
        process_transcription,
        recording_id,
        str(file_path),
        config,
    )

    return recording


async def process_transcription(
    recording_id: str,
    audio_path: str,
    config: TranscriptionConfig,
):
    """Background task to process transcription and diarization."""
    from ..models.database import async_session, Recording, Segment

    async with async_session() as db:
        try:
            # Update status to processing
            stmt = select(Recording).where(Recording.id == recording_id)
            result = await db.execute(stmt)
            recording = result.scalar_one()
            recording.status = RecordingStatus.PROCESSING.value
            await db.commit()

            # Initialize services
            transcription_svc = get_transcription_service()
            diarization_svc = get_diarization_service()
            speaker_learning_svc = get_speaker_learning_service()

            # Step 1: Diarization (if enabled)
            diarization_result = None
            if config.enable_diarization:
                try:
                    diarization_result = await diarization_svc.diarize(audio_path)
                except Exception as e:
                    print(f"Diarization failed, continuing without: {e}")

            # Step 2: Transcription
            transcription_result = await transcription_svc.transcribe(
                audio_path,
                language=config.language,
            )

            # Update recording duration
            recording.duration_ms = transcription_result.duration_ms

            # Step 3: Align transcription with diarization
            segments = []
            if diarization_result and diarization_result.segments:
                # Merge transcription segments with speaker labels
                segments = align_transcription_with_diarization(
                    transcription_result.segments,
                    diarization_result.segments,
                )
            else:
                # No diarization - use transcription segments as-is
                for seg in transcription_result.segments:
                    segments.append({
                        "text": seg.text,
                        "start_ms": seg.start_ms,
                        "end_ms": seg.end_ms,
                        "confidence": seg.confidence,
                        "temp_speaker_id": "Speaker A",
                        "words": seg.words,
                    })

            # Step 4: Try to match with known speakers
            if config.enable_speaker_matching:
                profiles = await speaker_learning_svc.load_profiles(db)
                if profiles:
                    from ..services.speaker_learning import SpeakerMatcher
                    matcher = SpeakerMatcher(profiles)

                    for seg in segments:
                        # Extract embedding for this segment if possible
                        # (This would require loading audio, simplified here)
                        pass

            # Step 5: Save segments to database
            for seg in segments:
                segment = Segment(
                    id=str(uuid.uuid4()),
                    recording_id=recording_id,
                    text=seg["text"],
                    start_ms=seg["start_ms"],
                    end_ms=seg["end_ms"],
                    confidence=seg.get("confidence"),
                    temp_speaker_id=seg.get("temp_speaker_id"),
                    word_timestamps=seg.get("words"),
                )
                db.add(segment)

            # Mark as completed
            recording.status = RecordingStatus.COMPLETED.value
            await db.commit()

        except Exception as e:
            # Mark as failed
            stmt = select(Recording).where(Recording.id == recording_id)
            result = await db.execute(stmt)
            recording = result.scalar_one()
            recording.status = RecordingStatus.FAILED.value
            recording.error_message = str(e)
            await db.commit()
            raise


def align_transcription_with_diarization(transcription_segments, diarization_segments):
    """
    Align transcription segments with speaker diarization.

    For each transcription segment, find the overlapping diarization segment
    and assign the speaker label.
    """
    aligned = []

    for trans_seg in transcription_segments:
        trans_start = trans_seg.start_ms
        trans_end = trans_seg.end_ms
        trans_mid = (trans_start + trans_end) / 2

        # Find best matching diarization segment
        best_speaker = "Speaker A"
        best_overlap = 0

        for diar_seg in diarization_segments:
            # Calculate overlap
            overlap_start = max(trans_start, diar_seg.start_ms)
            overlap_end = min(trans_end, diar_seg.end_ms)
            overlap = max(0, overlap_end - overlap_start)

            if overlap > best_overlap:
                best_overlap = overlap
                best_speaker = diar_seg.speaker_id

        aligned.append({
            "text": trans_seg.text,
            "start_ms": trans_start,
            "end_ms": trans_end,
            "confidence": trans_seg.confidence,
            "temp_speaker_id": best_speaker,
            "words": trans_seg.words,
        })

    return aligned


@router.get("/{recording_id}", response_model=RecordingWithSegments)
async def get_recording(
    recording_id: str,
    db: AsyncSession = Depends(get_db),
):
    """Get recording details with all segments and speaker summary."""
    stmt = (
        select(Recording)
        .where(Recording.id == recording_id)
        .options(selectinload(Recording.segments).selectinload(Segment.speaker))
    )
    result = await db.execute(stmt)
    recording = result.scalar_one_or_none()

    if not recording:
        raise HTTPException(status_code=404, detail="Recording not found")

    # Build speaker summary
    speaker_stats = {}
    for seg in recording.segments:
        speaker_key = seg.speaker_id or seg.temp_speaker_id or "Unknown"
        speaker_name = seg.speaker.name if seg.speaker else seg.temp_speaker_id or "Unknown"
        speaker_color = seg.speaker.color if seg.speaker else None

        if speaker_key not in speaker_stats:
            speaker_stats[speaker_key] = {
                "speaker_id": seg.speaker_id,
                "speaker_name": speaker_name,
                "temp_speaker_id": seg.temp_speaker_id if not seg.speaker_id else None,
                "color": speaker_color,
                "total_duration_ms": 0,
                "segment_count": 0,
                "word_count": 0,
            }

        stats = speaker_stats[speaker_key]
        stats["total_duration_ms"] += seg.end_ms - seg.start_ms
        stats["segment_count"] += 1
        stats["word_count"] += len(seg.text.split())

    # Calculate percentages
    total_duration = sum(s["total_duration_ms"] for s in speaker_stats.values())
    speaker_summary = []
    for key, stats in speaker_stats.items():
        stats["percentage"] = (
            (stats["total_duration_ms"] / total_duration * 100)
            if total_duration > 0
            else 0
        )
        speaker_summary.append(SpeakerSummary(**stats))

    # Sort by duration descending
    speaker_summary.sort(key=lambda s: s.total_duration_ms, reverse=True)

    # Build response
    response = RecordingWithSegments(
        id=recording.id,
        title=recording.title,
        description=recording.description,
        source_type=recording.source_type,
        source_path=recording.source_path,
        duration_ms=recording.duration_ms,
        status=recording.status,
        audio_format=recording.audio_format,
        created_at=recording.created_at,
        updated_at=recording.updated_at,
        segments=[
            SegmentResponse(
                id=seg.id,
                recording_id=seg.recording_id,
                speaker_id=seg.speaker_id,
                temp_speaker_id=seg.temp_speaker_id,
                text=seg.text,
                start_ms=seg.start_ms,
                end_ms=seg.end_ms,
                confidence=seg.confidence,
                speaker_confidence=seg.speaker_confidence,
                is_user_verified=seg.is_user_verified,
                word_timestamps=seg.word_timestamps,
                speaker=seg.speaker,
                created_at=seg.created_at,
            )
            for seg in sorted(recording.segments, key=lambda s: s.start_ms)
        ],
        speaker_summary=speaker_summary,
    )

    return response


@router.delete("/{recording_id}")
async def delete_recording(
    recording_id: str,
    db: AsyncSession = Depends(get_db),
):
    """Delete a recording and all its segments."""
    stmt = select(Recording).where(Recording.id == recording_id)
    result = await db.execute(stmt)
    recording = result.scalar_one_or_none()

    if not recording:
        raise HTTPException(status_code=404, detail="Recording not found")

    # Delete audio file if it exists
    if recording.source_path and os.path.exists(recording.source_path):
        os.remove(recording.source_path)

    # Delete recording (cascades to segments)
    await db.delete(recording)
    await db.commit()

    return {"status": "deleted"}


@router.post("/{recording_id}/export")
async def export_recording(
    recording_id: str,
    request: ExportRequest,
    db: AsyncSession = Depends(get_db),
):
    """Export transcript in the specified format."""
    # Get recording with segments
    stmt = (
        select(Recording)
        .where(Recording.id == recording_id)
        .options(selectinload(Recording.segments).selectinload(Segment.speaker))
    )
    result = await db.execute(stmt)
    recording = result.scalar_one_or_none()

    if not recording:
        raise HTTPException(status_code=404, detail="Recording not found")

    # Prepare segment data
    segments = [
        {
            "id": seg.id,
            "text": seg.text,
            "start_ms": seg.start_ms,
            "end_ms": seg.end_ms,
            "speaker_id": seg.speaker_id,
            "speaker_name": seg.speaker.name if seg.speaker else None,
            "temp_speaker_id": seg.temp_speaker_id,
            "confidence": seg.confidence,
            "word_timestamps": seg.word_timestamps,
        }
        for seg in sorted(recording.segments, key=lambda s: s.start_ms)
    ]

    # Prepare recording data
    recording_data = {
        "id": recording.id,
        "title": recording.title,
        "description": recording.description,
        "duration_ms": recording.duration_ms,
        "created_at": recording.created_at.isoformat() if recording.created_at else None,
    }

    # Get unique speakers
    speakers = []
    seen_speakers = set()
    for seg in recording.segments:
        if seg.speaker and seg.speaker.id not in seen_speakers:
            speakers.append({
                "id": seg.speaker.id,
                "name": seg.speaker.name,
                "color": seg.speaker.color,
            })
            seen_speakers.add(seg.speaker.id)

    # Export
    content, filename = _export_service.export(
        format=request.format,
        recording=recording_data,
        segments=segments,
        speakers=speakers,
        include_timestamps=request.include_timestamps,
        include_speaker_names=request.include_speaker_names,
    )

    # Determine content type
    content_types = {
        ExportFormat.TXT: "text/plain",
        ExportFormat.SRT: "application/x-subrip",
        ExportFormat.VTT: "text/vtt",
        ExportFormat.JSON: "application/json",
    }
    content_type = content_types.get(request.format, "text/plain")

    return Response(
        content=content,
        media_type=content_type,
        headers={
            "Content-Disposition": f'attachment; filename="{filename}"',
        },
    )
