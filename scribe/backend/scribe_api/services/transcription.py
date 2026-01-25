"""Transcription service using OpenAI Whisper API."""

import asyncio
import os
import tempfile
from pathlib import Path
from typing import AsyncGenerator, List, Optional, Callable
from dataclasses import dataclass
import logging

import numpy as np
import soundfile as sf
from pydub import AudioSegment

logger = logging.getLogger(__name__)


@dataclass
class TranscriptionSegment:
    """A transcribed segment with timing information."""
    text: str
    start_ms: int
    end_ms: int
    confidence: float
    words: Optional[List[dict]] = None  # Word-level timestamps


@dataclass
class TranscriptionResult:
    """Complete transcription result."""
    segments: List[TranscriptionSegment]
    language: str
    duration_ms: int


class TranscriptionService:
    """Service for transcribing audio using Whisper."""

    def __init__(
        self,
        api_key: Optional[str] = None,
        use_local: bool = False,
        local_model: str = "large-v3",
    ):
        self.api_key = api_key or os.getenv("OPENAI_API_KEY")
        self.use_local = use_local
        self.local_model = local_model
        self._local_model_instance = None

        if not self.use_local and not self.api_key:
            raise ValueError("OpenAI API key required for cloud transcription")

    async def transcribe(
        self,
        audio_path: str,
        language: str = "auto",
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> TranscriptionResult:
        """
        Transcribe an audio file.

        Args:
            audio_path: Path to the audio file
            language: Language code or "auto" for detection
            on_progress: Optional callback for progress updates

        Returns:
            TranscriptionResult with segments and metadata
        """
        # Convert to WAV if needed
        wav_path = await self._ensure_wav(audio_path, on_progress)

        try:
            if self.use_local:
                return await self._transcribe_local(wav_path, language, on_progress)
            else:
                return await self._transcribe_api(wav_path, language, on_progress)
        finally:
            # Clean up temp file if we created one
            if wav_path != audio_path and os.path.exists(wav_path):
                os.remove(wav_path)

    async def _ensure_wav(
        self,
        audio_path: str,
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> str:
        """Convert audio to WAV format if needed."""
        path = Path(audio_path)

        if path.suffix.lower() == ".wav":
            return audio_path

        if on_progress:
            on_progress(5, "Converting audio format...")

        # Use pydub for conversion
        def convert():
            audio = AudioSegment.from_file(audio_path)
            # Convert to 16kHz mono for optimal Whisper performance
            audio = audio.set_frame_rate(16000).set_channels(1)

            temp_path = tempfile.mktemp(suffix=".wav")
            audio.export(temp_path, format="wav")
            return temp_path

        wav_path = await asyncio.get_event_loop().run_in_executor(None, convert)
        return wav_path

    async def _transcribe_api(
        self,
        wav_path: str,
        language: str,
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> TranscriptionResult:
        """Transcribe using OpenAI Whisper API."""
        import httpx

        if on_progress:
            on_progress(10, "Uploading to OpenAI...")

        # Read audio file
        with open(wav_path, "rb") as f:
            audio_data = f.read()

        # Get audio duration
        info = sf.info(wav_path)
        duration_ms = int(info.duration * 1000)

        # Prepare API request
        headers = {
            "Authorization": f"Bearer {self.api_key}",
        }

        files = {
            "file": ("audio.wav", audio_data, "audio/wav"),
            "model": (None, "whisper-1"),
            "response_format": (None, "verbose_json"),
            "timestamp_granularities[]": (None, "word"),
            "timestamp_granularities[]": (None, "segment"),
        }

        if language != "auto":
            files["language"] = (None, language)

        if on_progress:
            on_progress(30, "Transcribing with Whisper...")

        async with httpx.AsyncClient(timeout=300.0) as client:
            response = await client.post(
                "https://api.openai.com/v1/audio/transcriptions",
                headers=headers,
                files=files,
            )
            response.raise_for_status()
            result = response.json()

        if on_progress:
            on_progress(90, "Processing results...")

        # Parse response
        segments = []
        for seg in result.get("segments", []):
            words = None
            if "words" in result:
                # Filter words for this segment
                seg_words = [
                    w for w in result["words"]
                    if seg["start"] <= w["start"] < seg["end"]
                ]
                words = [
                    {
                        "word": w["word"],
                        "start_ms": int(w["start"] * 1000),
                        "end_ms": int(w["end"] * 1000),
                    }
                    for w in seg_words
                ]

            segments.append(TranscriptionSegment(
                text=seg["text"].strip(),
                start_ms=int(seg["start"] * 1000),
                end_ms=int(seg["end"] * 1000),
                confidence=seg.get("avg_logprob", 0.0),
                words=words,
            ))

        return TranscriptionResult(
            segments=segments,
            language=result.get("language", "unknown"),
            duration_ms=duration_ms,
        )

    async def _transcribe_local(
        self,
        wav_path: str,
        language: str,
        on_progress: Optional[Callable[[float, str], None]] = None,
    ) -> TranscriptionResult:
        """Transcribe using local faster-whisper model."""
        if on_progress:
            on_progress(10, "Loading Whisper model...")

        # Lazy load the model
        if self._local_model_instance is None:
            from faster_whisper import WhisperModel

            # Use MPS on Mac if available
            device = "cuda" if os.getenv("CUDA_VISIBLE_DEVICES") else "cpu"
            compute_type = "float16" if device == "cuda" else "int8"

            self._local_model_instance = WhisperModel(
                self.local_model,
                device=device,
                compute_type=compute_type,
            )

        if on_progress:
            on_progress(30, "Transcribing locally...")

        # Get audio duration
        info = sf.info(wav_path)
        duration_ms = int(info.duration * 1000)

        # Run transcription
        def run_transcription():
            lang = None if language == "auto" else language
            segments_gen, info = self._local_model_instance.transcribe(
                wav_path,
                language=lang,
                word_timestamps=True,
                vad_filter=True,
            )
            return list(segments_gen), info.language

        segments_raw, detected_lang = await asyncio.get_event_loop().run_in_executor(
            None, run_transcription
        )

        if on_progress:
            on_progress(90, "Processing results...")

        # Parse segments
        segments = []
        for seg in segments_raw:
            words = None
            if seg.words:
                words = [
                    {
                        "word": w.word,
                        "start_ms": int(w.start * 1000),
                        "end_ms": int(w.end * 1000),
                    }
                    for w in seg.words
                ]

            segments.append(TranscriptionSegment(
                text=seg.text.strip(),
                start_ms=int(seg.start * 1000),
                end_ms=int(seg.end * 1000),
                confidence=seg.avg_logprob,
                words=words,
            ))

        return TranscriptionResult(
            segments=segments,
            language=detected_lang,
            duration_ms=duration_ms,
        )

    async def transcribe_segment(
        self,
        audio_data: np.ndarray,
        sample_rate: int = 16000,
        language: str = "auto",
    ) -> Optional[str]:
        """Transcribe a single audio segment (for real-time use)."""
        # Save to temp file
        temp_path = tempfile.mktemp(suffix=".wav")
        try:
            sf.write(temp_path, audio_data, sample_rate)
            result = await self.transcribe(temp_path, language)
            if result.segments:
                return " ".join(s.text for s in result.segments)
            return None
        finally:
            if os.path.exists(temp_path):
                os.remove(temp_path)
