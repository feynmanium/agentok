"""Export service for transcripts."""

from typing import List, Optional
from datetime import timedelta
import json

from ..models.schemas import ExportFormat


def ms_to_srt_timestamp(ms: int) -> str:
    """Convert milliseconds to SRT timestamp format (HH:MM:SS,mmm)."""
    td = timedelta(milliseconds=ms)
    hours = int(td.total_seconds() // 3600)
    minutes = int((td.total_seconds() % 3600) // 60)
    seconds = int(td.total_seconds() % 60)
    milliseconds = ms % 1000
    return f"{hours:02d}:{minutes:02d}:{seconds:02d},{milliseconds:03d}"


def ms_to_vtt_timestamp(ms: int) -> str:
    """Convert milliseconds to VTT timestamp format (HH:MM:SS.mmm)."""
    td = timedelta(milliseconds=ms)
    hours = int(td.total_seconds() // 3600)
    minutes = int((td.total_seconds() % 3600) // 60)
    seconds = int(td.total_seconds() % 60)
    milliseconds = ms % 1000
    return f"{hours:02d}:{minutes:02d}:{seconds:02d}.{milliseconds:03d}"


def ms_to_simple_timestamp(ms: int) -> str:
    """Convert milliseconds to simple timestamp (MM:SS)."""
    td = timedelta(milliseconds=ms)
    total_seconds = int(td.total_seconds())
    minutes = total_seconds // 60
    seconds = total_seconds % 60
    return f"{minutes:02d}:{seconds:02d}"


class ExportService:
    """Service for exporting transcripts in various formats."""

    def export_txt(
        self,
        segments: List[dict],
        include_timestamps: bool = True,
        include_speaker_names: bool = True,
    ) -> str:
        """
        Export transcript as plain text.

        Args:
            segments: List of segment dictionaries with text, speaker, start_ms, end_ms
            include_timestamps: Whether to include timestamps
            include_speaker_names: Whether to include speaker names

        Returns:
            Plain text transcript
        """
        lines = []
        current_speaker = None

        for seg in segments:
            speaker_name = seg.get("speaker_name") or seg.get("temp_speaker_id", "Unknown")
            text = seg.get("text", "").strip()

            if not text:
                continue

            # Build line
            parts = []

            if include_timestamps:
                timestamp = ms_to_simple_timestamp(seg.get("start_ms", 0))
                parts.append(f"[{timestamp}]")

            if include_speaker_names:
                # Only show speaker name when it changes
                if speaker_name != current_speaker:
                    parts.append(f"{speaker_name}:")
                    current_speaker = speaker_name
                else:
                    # Indent continuation
                    if include_timestamps:
                        parts.append("  ")

            parts.append(text)
            lines.append(" ".join(parts))

        return "\n".join(lines)

    def export_srt(
        self,
        segments: List[dict],
        include_speaker_names: bool = True,
    ) -> str:
        """
        Export transcript as SRT subtitle file.

        Args:
            segments: List of segment dictionaries
            include_speaker_names: Whether to include speaker names in subtitles

        Returns:
            SRT formatted string
        """
        lines = []

        for i, seg in enumerate(segments, 1):
            text = seg.get("text", "").strip()
            if not text:
                continue

            start = ms_to_srt_timestamp(seg.get("start_ms", 0))
            end = ms_to_srt_timestamp(seg.get("end_ms", 0))

            # Add speaker name as prefix if requested
            if include_speaker_names:
                speaker_name = seg.get("speaker_name") or seg.get("temp_speaker_id", "")
                if speaker_name:
                    text = f"[{speaker_name}] {text}"

            lines.append(str(i))
            lines.append(f"{start} --> {end}")
            lines.append(text)
            lines.append("")  # Blank line between entries

        return "\n".join(lines)

    def export_vtt(
        self,
        segments: List[dict],
        include_speaker_names: bool = True,
    ) -> str:
        """
        Export transcript as WebVTT subtitle file.

        Args:
            segments: List of segment dictionaries
            include_speaker_names: Whether to include speaker names

        Returns:
            VTT formatted string
        """
        lines = ["WEBVTT", ""]  # VTT header

        for seg in segments:
            text = seg.get("text", "").strip()
            if not text:
                continue

            start = ms_to_vtt_timestamp(seg.get("start_ms", 0))
            end = ms_to_vtt_timestamp(seg.get("end_ms", 0))

            # Add speaker as voice tag if requested
            if include_speaker_names:
                speaker_name = seg.get("speaker_name") or seg.get("temp_speaker_id", "")
                if speaker_name:
                    text = f"<v {speaker_name}>{text}"

            lines.append(f"{start} --> {end}")
            lines.append(text)
            lines.append("")

        return "\n".join(lines)

    def export_json(
        self,
        recording: dict,
        segments: List[dict],
        speakers: List[dict],
    ) -> str:
        """
        Export transcript as JSON.

        Args:
            recording: Recording metadata
            segments: List of segment dictionaries
            speakers: List of speaker dictionaries

        Returns:
            JSON formatted string
        """
        export_data = {
            "recording": {
                "id": recording.get("id"),
                "title": recording.get("title"),
                "description": recording.get("description"),
                "duration_ms": recording.get("duration_ms"),
                "created_at": recording.get("created_at"),
            },
            "speakers": [
                {
                    "id": s.get("id"),
                    "name": s.get("name"),
                    "color": s.get("color"),
                }
                for s in speakers
            ],
            "segments": [
                {
                    "id": seg.get("id"),
                    "speaker_id": seg.get("speaker_id"),
                    "speaker_name": seg.get("speaker_name") or seg.get("temp_speaker_id"),
                    "text": seg.get("text"),
                    "start_ms": seg.get("start_ms"),
                    "end_ms": seg.get("end_ms"),
                    "confidence": seg.get("confidence"),
                    "word_timestamps": seg.get("word_timestamps"),
                }
                for seg in segments
            ],
        }

        return json.dumps(export_data, indent=2, default=str)

    def export(
        self,
        format: ExportFormat,
        recording: dict,
        segments: List[dict],
        speakers: Optional[List[dict]] = None,
        include_timestamps: bool = True,
        include_speaker_names: bool = True,
    ) -> tuple[str, str]:
        """
        Export transcript in the specified format.

        Args:
            format: Export format (txt, srt, vtt, json)
            recording: Recording metadata
            segments: List of segment dictionaries
            speakers: List of speaker dictionaries (for JSON export)
            include_timestamps: Whether to include timestamps (for TXT)
            include_speaker_names: Whether to include speaker names

        Returns:
            Tuple of (content, filename)
        """
        title = recording.get("title", "transcript")
        safe_title = "".join(c if c.isalnum() or c in " -_" else "_" for c in title)

        if format == ExportFormat.TXT:
            content = self.export_txt(segments, include_timestamps, include_speaker_names)
            filename = f"{safe_title}.txt"
        elif format == ExportFormat.SRT:
            content = self.export_srt(segments, include_speaker_names)
            filename = f"{safe_title}.srt"
        elif format == ExportFormat.VTT:
            content = self.export_vtt(segments, include_speaker_names)
            filename = f"{safe_title}.vtt"
        elif format == ExportFormat.JSON:
            content = self.export_json(recording, segments, speakers or [])
            filename = f"{safe_title}.json"
        else:
            raise ValueError(f"Unsupported export format: {format}")

        return content, filename
